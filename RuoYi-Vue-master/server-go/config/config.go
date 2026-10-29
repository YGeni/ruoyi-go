// Package config 负责加载 server-go 的运行期配置。
//
// 配置项与 ruoyi-admin/src/main/resources/application.yml、
// application-druid.yml 一一对应，Redis key 前缀沿用 CacheConstants 的取值
// （见 constant.go），保证与 Java 端共存时读的是同一份缓存。
package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// C 为全局配置，启动时由 Load 填充。
var C Config

// Config 对应 config/config.yaml 的完整结构。
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Spring     SpringConfig     `mapstructure:"spring"`
	RuoYi      RuoYiConfig      `mapstructure:"ruoyi"`
	User       UserConfig       `mapstructure:"user"`
	Datasource DatasourceConfig `mapstructure:"datasource"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Token      TokenConfig      `mapstructure:"token"`
	Log        LogConfig        `mapstructure:"log"`
	Xss        XssConfig        `mapstructure:"xss"`
	Referer    RefererConfig    `mapstructure:"referer"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// SpringConfig 目前只需 multipart 上传限制（application.yml: spring.servlet.multipart）。
type SpringConfig struct {
	Servlet ServletConfig `mapstructure:"servlet"`
}

type ServletConfig struct {
	Multipart MultipartConfig `mapstructure:"multipart"`
}

// MultipartConfig 等价 Spring Boot 的 MultipartProperties。
//
// 值形如 "10MB / 20MB"，解析见 ParseDataSize（config.go）。
type MultipartConfig struct {
	MaxFileSize    string `mapstructure:"max-file-size"`
	MaxRequestSize string `mapstructure:"max-request-size"`
}

// MaxFileSizeBytes 返回单文件上限字节数，解析失败回退 10MB。
func (m MultipartConfig) MaxFileSizeBytes() int64 {
	return orDefault(ParseDataSize(m.MaxFileSize), 10*1024*1024)
}

// MaxRequestSizeBytes 返回总请求上限字节数，解析失败回退 20MB。
func (m MultipartConfig) MaxRequestSizeBytes() int64 {
	return orDefault(ParseDataSize(m.MaxRequestSize), 20*1024*1024)
}

func orDefault(v int64, def int64) int64 {
	if v <= 0 {
		return def
	}
	return v
}

type RuoYiConfig struct {
	Name           string `mapstructure:"name"`
	Version        string `mapstructure:"version"`
	Profile        string `mapstructure:"profile"`
	AddressEnabled bool   `mapstructure:"addressEnabled"`
	CaptchaType    string `mapstructure:"captchaType"`
}

type UserConfig struct {
	Password PasswordConfig `mapstructure:"password"`
}

type PasswordConfig struct {
	MaxRetryCount int `mapstructure:"maxRetryCount"`
	LockTime      int `mapstructure:"lockTime"` // 单位：分钟
}

type DatasourceConfig struct {
	Master DBConfig  `mapstructure:"master"`
	Slave  SlaveConf `mapstructure:"slave"`
}

type DBConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	Database    string `mapstructure:"database"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	Params      string `mapstructure:"params"`
	InitialSize int    `mapstructure:"initialSize"`
	MinIdle     int    `mapstructure:"minIdle"`
	MaxActive   int    `mapstructure:"maxActive"`
	MaxWait     int    `mapstructure:"maxWait"` // 单位：毫秒
}

type SlaveConf struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database int    `mapstructure:"database"`
	Password string `mapstructure:"password"`
}

type TokenConfig struct {
	Header     string `mapstructure:"header"`
	Secret     string `mapstructure:"secret"`
	ExpireTime int    `mapstructure:"expireTime"` // 单位：分钟
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

type XssConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Excludes    string `mapstructure:"excludes"`
	URLPatterns string `mapstructure:"urlPatterns"`
}

type RefererConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	AllowedDomains string `mapstructure:"allowedDomains"`
}

// DSN 组装 MySQL 连接串。
// DIFF(review): Java 侧 jdbc url 带 serverTimezone=GMT%2B8，Go 侧用 loc=Local
// 依赖容器/本机时区（docker-compose 已固定 --default-time-zone=+08:00），语义等价。
func (c DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		c.Username, c.Password, c.Host, c.Port, c.Database, c.Params)
}

// Addr 组装 host:port。
func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// TokenExpireDuration 令牌有效期。
func (c TokenConfig) TokenExpireDuration() time.Duration {
	return time.Duration(c.ExpireTime) * time.Minute
}

// LockTimeDuration 密码锁定时间。
func (c PasswordConfig) LockTimeDuration() time.Duration {
	return time.Duration(c.LockTime) * time.Minute
}

// Load 读取配置文件并填充全局 C。
func Load(path string) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置失败: %w", err)
	}
	if err := v.Unmarshal(&C); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}
	applyDefaults()
	// ruoyi.profile 支持 ${ENV:默认值} 占位（viper 本身不解析该语法）。
	// 原 application.yml 是 Windows 路径 D:/ruoyi/uploadPath，在 macOS/Linux 下
	// 会在工作目录生成名为 "D:" 的目录，故默认改为仓库内相对路径并允许环境变量覆盖。
	C.RuoYi.Profile = expandEnvDefault(C.RuoYi.Profile)
	return nil
}

// envDefaultPattern 匹配 ${VAR} 或 ${VAR:默认值}。
var envDefaultPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::([^{}]*))?\}`)

// expandEnvDefault 把 ${VAR:默认值} 展开为环境变量值；未设置（或为空）时回退默认值。
func expandEnvDefault(s string) string {
	return envDefaultPattern.ReplaceAllStringFunc(s, func(m string) string {
		sub := envDefaultPattern.FindStringSubmatch(m)
		name, def := sub[1], sub[2]
		if v := os.Getenv(name); v != "" {
			return v
		}
		return def
	})
}

// ---------- RuoYiConfig 的路径派生（ruoyi-----------
//
// 四个派生目录一律以 getProfile() 为根，Java 侧是字符串拼接（含尾斜杠差异），
// Go 侧逐字照搬，避免下载时路径对不上。

func (r RuoYiConfig) GetProfile() string    { return r.Profile }
func (r RuoYiConfig) GetImportPath() string { return r.Profile + "/import" }
func (r RuoYiConfig) GetAvatarPath() string { return r.Profile + "/avatar" }
func (r RuoYiConfig) GetUploadPath() string { return r.Profile + "/upload" }

// GetDownloadPath 注意 Java 侧带**尾斜杠**
// /common/download 的 filePath = getDownloadPath() + fileName，
// 少了这个斜杠会拼出 "…/downloadtest.png"。
func (r RuoYiConfig) GetDownloadPath() string { return r.Profile + "/download/" }

// ParseDataSize 解析 Spring Boot 的 DataSize 字面量（"10MB"、"512KB"、"1GB"、裸字节数）。
//
// 只支持 Spring 支持的 B/KB/MB/GB/TB（大小写不敏感），解析失败返回 0 由调用方兜底。
func ParseDataSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// 纯数字按字节处理（Spring 的 DataSize 语义）
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	upper := strings.ToUpper(s)
	units := []struct {
		suffix string
		mul    int64
	}{
		{"TB", 1024 * 1024 * 1024 * 1024},
		{"GB", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}
	for _, u := range units {
		if strings.HasSuffix(upper, u.suffix) {
			num := strings.TrimSpace(upper[:len(upper)-len(u.suffix)])
			if n, err := strconv.ParseInt(num, 10, 64); err == nil {
				return n * u.mul
			}
			return 0
		}
	}
	return 0
}

func applyDefaults() {
	if C.Server.Port == 0 {
		C.Server.Port = 8080
	}
	if C.Server.Mode == "" {
		C.Server.Mode = "debug"
	}
	if C.RuoYi.CaptchaType == "" {
		C.RuoYi.CaptchaType = "math"
	}
	if C.Token.Header == "" {
		C.Token.Header = "Authorization"
	}
	if C.Token.ExpireTime == 0 {
		C.Token.ExpireTime = 30
	}
	if C.User.Password.MaxRetryCount == 0 {
		C.User.Password.MaxRetryCount = 5
	}
	if C.User.Password.LockTime == 0 {
		C.User.Password.LockTime = 10
	}
}
