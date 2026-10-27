// Package redisx 封装 go-redis 客户端，key 前缀沿用 CacheConstants。
package redisx

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"

	"github.com/redis/go-redis/v9"
)

var client *redis.Client

// Init 初始化 Redis 客户端。
func Init(cfg config.RedisConfig) error {
	client = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.Database,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return client.Ping(ctx).Err()
}

// Client 返回底层客户端。
func Client() *redis.Client { return client }

// Close 关闭连接。
func Close() {
	if client != nil {
		_ = client.Close()
	}
}

// Set 写入字符串键值。
func Set(key string, value any, ttl time.Duration) error {
	return client.Set(context.Background(), key, value, ttl).Err()
}

// Get 读取字符串；ok=false 表示键不存在。
func Get(key string) (string, bool, error) {
	v, err := client.Get(context.Background(), key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

// Del 删除键。
func Del(key string) error {
	return client.Del(context.Background(), key).Err()
}

// HasKey 判断键是否存在。
func HasKey(key string) (bool, error) {
	n, err := client.Exists(context.Background(), key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Keys 按 pattern 扫描键（对应 Java 的 redisCache.keys(pattern)）。
func Keys(pattern string) ([]string, error) {
	return client.Keys(context.Background(), pattern).Result()
}

// Expire 设置过期时间。
func Expire(key string, ttl time.Duration) error {
	return client.Expire(context.Background(), key, ttl).Err()
}

// SetObject 序列化为 JSON 后写入。
func SetObject(key string, v any, ttl time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return Set(key, b, ttl)
}

// GetObject 读取 JSON 并反序列化；ok=false 表示键不存在。
func GetObject(key string, dest any) (bool, error) {
	s, ok, err := Get(key)
	if err != nil || !ok {
		return false, err
	}
	if err := json.Unmarshal([]byte(s), dest); err != nil {
		return false, err
	}
	return true, nil
}

// Incr 自增并返回新值（用于密码错误计数）。
func Incr(key string) (int64, error) {
	return client.Incr(context.Background(), key).Result()
}
