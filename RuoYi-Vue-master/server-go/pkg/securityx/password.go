// Package securityx 复刻密码校验与登录失败计数。
//
// 对应实现：
// -ruoyi-
//
//	计数键 pwd_err_cnt:<username>（L39-42），达到 maxRetryCount(5) 直接拒绝（L57-60），
//	失败则自增并设 lockTime(10 分钟) 过期（L62-67），成功则清除计数（L79-85）
//
// -ruoyi-BCrypt
package securityx

import (
	"errors"
	"fmt"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/pkg/redisx"

	"golang.org/x/crypto/bcrypt"
)

// ErrPasswordNotMatch 密码不匹配（提示文案与 Java 的 user.password.not.match 一致）。
var ErrPasswordNotMatch = errors.New(config.MsgUserNotExists)

// ErrPasswordRetryExceed 密码错误次数超限。
var ErrPasswordRetryExceed = errors.New(config.MsgUserNotExists)

func cacheKey(username string) string {
	return config.PwdErrCntKey + username
}

// Matches 校验明文密码与 BCrypt 散列是否匹配。
func Matches(rawPassword, hashed string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(rawPassword)) == nil
}

// Encrypt 生成 BCrypt 散列（对应 SecurityUtils.encryptPassword）。
func Encrypt(rawPassword string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Validate 校验密码并处理失败计数。会先判断是否已锁定，再比对密码。
func Validate(username, rawPassword, hashed string) error {
	retryCount, err := currentRetryCount(username)
	if err != nil {
		return err
	}
	if retryCount >= int64(config.C.User.Password.MaxRetryCount) {
		// 与 UserPasswordRetryLimitExceedException 文案一致
		return fmt.Errorf(config.MsgPasswordRetryExceed,
			config.C.User.Password.MaxRetryCount, config.C.User.Password.LockTime)
	}
	if !Matches(rawPassword, hashed) {
		if _, err := redisx.Incr(cacheKey(username)); err != nil {
			return err
		}
		// 计数键的过期时间即锁定时长
		if err := redisx.Expire(cacheKey(username), config.C.User.Password.LockTimeDuration()); err != nil {
			return err
		}
		return ErrPasswordNotMatch
	}
	return ClearLoginRecordCache(username)
}

// ClearLoginRecordCache 清除失败计数。
func ClearLoginRecordCache(username string) error {
	ok, err := redisx.HasKey(cacheKey(username))
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return redisx.Del(cacheKey(username))
}

func currentRetryCount(username string) (int64, error) {
	s, ok, err := redisx.Get(cacheKey(username))
	if err != nil {
		return 0, err
	}
	if !ok || s == "" {
		return 0, nil
	}
	var n int64
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0, nil
	}
	return n, nil
}
