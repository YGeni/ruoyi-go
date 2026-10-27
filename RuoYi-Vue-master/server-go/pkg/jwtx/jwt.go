// Package jwtx 复刻 TokenService 的令牌签发 / 解析 / 刷新逻辑。
//
// 对应实现：ruoyi-framework/src/main/java/com/ruoyi/framework/web/service/TokenService.java
//   - createToken(L115-126)：uuid 作为 token 主键，claims = {login_user_key: uuid, sub: username}
//   - createToken(L179-185)：HS512 签名，密钥取 token.secret
//   - getToken(L219-227)：剥离 "Bearer " 前缀（前端恒定带前缀，见前端 request.js:32）
//   - verifyToken(L134-142)：有效期不足 20 分钟自动刷新
package jwtx

import (
	"fmt"
	"strings"

	"github.com/YGeni/ruoyi-go/server-go/config"

	"github.com/golang-jwt/jwt/v5"
)

// Generate 生成 JWT；uuid 为登录态主键（Redis key 后缀），username 写进 sub。
func Generate(uuid, username string) (string, error) {
	claims := jwt.MapClaims{
		config.LoginUserKey: uuid,
		config.JWTUsername:  username,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	return token.SignedString([]byte(config.C.Token.Secret))
}

// Parse 解析 JWT，返回 uuid 与用户名。
func Parse(tokenStr string) (uuid string, username string, err error) {
	claims := jwt.MapClaims{}
	if _, err = jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名算法: %s", t.Method.Alg())
		}
		return []byte(config.C.Token.Secret), nil
	}); err != nil {
		return "", "", err
	}
	uuid, _ = claims[config.LoginUserKey].(string)
	username, _ = claims[config.JWTUsername].(string)
	if uuid == "" {
		return "", "", fmt.Errorf("令牌缺少 %s 声明", config.LoginUserKey)
	}
	return uuid, username, nil
}

// StripBearer 剥离 "Bearer " 前缀。
//
// DIFF(review): Java 用的是 replace(TOKEN_PREFIX, "")（TokenService.java:224），
// 会替换字符串中所有匹配片段；Go 侧只剥离开头的 "Bearer "，语义更严格且更安全。
func StripBearer(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, config.TokenPrefix) {
		return strings.TrimSpace(strings.TrimPrefix(v, config.TokenPrefix))
	}
	return v
}
