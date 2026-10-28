package service

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/pkg/redisx"
)

// ---------- 缓存监控：对应 CacheController.java ----------

// ErrRedisNotReady Redis 未初始化。
var ErrRedisNotReady = errors.New("Redis 未初始化")

// SysCache 缓存项，等价 ruoyi-system/.../system/domain/SysCache.java。
//
// JSON 字段与前端对齐：
//   - list.vue:174 把 getNames 的结果直接当表格行（cacheName + remark）
//   - list.vue:234 把 getValue 的结果塞进表单（cacheName / cacheKey / cacheValue）
type SysCache struct {
	CacheName  string `json:"cacheName"`
	CacheKey   string `json:"cacheKey"`
	CacheValue string `json:"cacheValue"`
	Remark     string `json:"remark"`
}

// cacheNames Java 的实例初始化块（CacheController.java:36-44）：
// 七类缓存的 key 前缀与中文名。顺序照抄，前端表格就是这个顺序。
var cacheNames = []SysCache{
	{CacheName: config.LoginTokenKey, Remark: "用户信息"},
	{CacheName: config.SysConfigKey, Remark: "配置信息"},
	{CacheName: config.SysDictKey, Remark: "数据字典"},
	{CacheName: config.CaptchaCodeKey, Remark: "验证码"},
	{CacheName: config.RepeatSubmitKey, Remark: "防重提交"},
	{CacheName: config.RateLimitKey, Remark: "限流处理"},
	{CacheName: config.PwdErrCntKey, Remark: "密码错误次数"},
}

// CacheNames 缓存名列表，等价 CacheController.cache(L73-77)。
func CacheNames() []SysCache {
	out := make([]SysCache, len(cacheNames))
	copy(out, cacheNames)
	return out
}

// CacheInfo Redis 监控信息，等价 CacheController.getInfo(L52-70)。
//
// 返回结构必须是 {info, dbSize, commandStats}，前端 index.vue:80/95 直接读
// response.data.commandStats 画饼图、response.data.dbSize 显示键数量。
type CacheInfo struct {
	Info         map[string]string   `json:"info"`
	DBSize       int64               `json:"dbSize"`
	CommandStats []map[string]string `json:"commandStats"`
}

// RedisInfo 取 Redis 运行信息。
//
// DIFF(review): Java 的 connection.info() 返回 Properties（键值都在一张表里），
// Go 侧把 "key:value" 逐行拆成 map。行首为 '#' 的是小节标题，跳过。
func RedisInfo() (*CacheInfo, error) {
	if redisx.Client() == nil {
		return nil, ErrRedisNotReady
	}
	ctx := context.Background()
	infoRaw, err := redisx.Client().Info(ctx).Result()
	if err != nil {
		return nil, err
	}
	cmdRaw, err := redisx.Client().Info(ctx, "commandstats").Result()
	if err != nil {
		return nil, err
	}
	size, err := redisx.Client().DBSize(ctx).Result()
	if err != nil {
		return nil, err
	}
	out := &CacheInfo{
		Info:         parseInfo(infoRaw),
		DBSize:       size,
		CommandStats: parseCommandStats(cmdRaw),
	}
	return out, nil
}

// parseInfo 把 INFO 输出拆成 map。
func parseInfo(raw string) map[string]string {
	m := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		m[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return m
}

// parseCommandStats 解析 INFO commandstats。
//
// 每行形如 `cmdstat_get:calls=1234,usec=56,usec_per_call=0.04,rejected_calls=0,failed_calls=0`；
// Java 侧取 name = 去掉 "cmdstat_" 前缀，value = "calls=" 与 ",usec" 之间的串
// （CacheController.java:64-69）。
func parseCommandStats(raw string) []map[string]string {
	var out []map[string]string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "cmdstat_") {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		name := strings.TrimPrefix(k, "cmdstat_")
		value := ""
		if i := strings.Index(v, "calls="); i >= 0 {
			rest := v[i+len("calls="):]
			if j := strings.Index(rest, ",usec"); j >= 0 {
				value = rest[:j]
			} else {
				value = rest
			}
		}
		out = append(out, map[string]string{"name": name, "value": value})
	}
	// Java 用的是 commandStats.stringPropertyNames()，Properties 底层是 hashtable，
	// 顺序不定；这里按调用次数降序，饼图更可读（前端只按 value 画，不依赖顺序）。
	sort.Slice(out, func(i, j int) bool {
		vi, _ := strconv.ParseInt(out[i]["value"], 10, 64)
		vj, _ := strconv.ParseInt(out[j]["value"], 10, 64)
		return vi > vj
	})
	return out
}

// CacheKeys 某类缓存下的全部 key，等价 CacheController.getCacheKeys(L82-87)。
//
// Java 返回 TreeSet（升序）；Go 侧排序后返回，顺序一致。
func CacheKeys(cacheName string) ([]string, error) {
	keys, err := redisx.Keys(cacheName + "*")
	if err != nil {
		return nil, err
	}
	sort.Strings(keys)
	if keys == nil {
		keys = []string{}
	}
	return keys, nil
}

// CacheValue 取某个 key 的值，等价 CacheController.getCacheValue(L92-98)。
//
// 注意 Java 是 `opsForValue().get(cacheKey)`，cacheKey 是**完整 key**（含前缀），
// cacheName 原样塞进 SysCache 不做拼接。
func CacheValue(cacheName, cacheKey string) (*SysCache, error) {
	// redisx.Get 已经把 redis.Nil 归一成 (v, false, nil)，键不存在不是错误。
	v, _, err := redisx.Get(cacheKey)
	if err != nil {
		return nil, err
	}
	return &SysCache{CacheName: cacheName, CacheKey: cacheKey, CacheValue: v}, nil
}

// ClearCacheName 清某类缓存，等价 CacheController.clearCacheName(L104-110)。
func ClearCacheName(cacheName string) error {
	keys, err := redisx.Keys(cacheName + "*")
	if err != nil {
		return err
	}
	for _, k := range keys {
		if err := redisx.Del(k); err != nil {
			return err
		}
	}
	return nil
}

// ClearCacheKey 清单个 key，等价 CacheController.clearCacheKey(L116-121)。
func ClearCacheKey(cacheKey string) error {
	return redisx.Del(cacheKey)
}

// ClearCacheAll 清空整个库，等价 CacheController.clearCacheAll(L127-133)。
//
// 危险操作：会连登录态一起清掉（login_tokens:* 也在 "*" 里），
// 与 Java 行为一致（Java 同样 keys("*") 全删）。
func ClearCacheAll() error {
	keys, err := redisx.Keys("*")
	if err != nil {
		return err
	}
	for _, k := range keys {
		if err := redisx.Del(k); err != nil {
			return err
		}
	}
	return nil
}
