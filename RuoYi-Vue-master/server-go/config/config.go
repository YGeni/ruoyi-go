// Package config 负责加载 server-go 的运行期配置。
//
// 配置项与 ruoyi-admin/src/main/resources/application.yml、
// application-druid.yml 一一对应，Redis key 前缀沿用 CacheConstants 的取值
// （见 constant.go），保证与 Java 端共存时读的是同一份缓存。
package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// C 为全局配置，启动时由 Load 填充。
var C Config

// Config 对应 config/config.yaml 的完整结构。
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
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
	return nil
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
