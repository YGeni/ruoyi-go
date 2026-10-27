// Package middleware 对应 ruoyi-framework 的过滤器与安全链路。
package middleware

import (
	"fmt"
	"net/http"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/service"
	"github.com/YGeni/ruoyi-go/server-go/pkg/jwtx"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

const (
	// CtxLoginUser 登录态在 gin.Context 中的键。
	CtxLoginUser = "loginUser"
	// CtxPermission 当前接口所需的权限串，等价于 Java 的 PermissionContextHolder
	// （PermissionService.java:38/70 写入，供 @DataScope 读取）。
	//
	// 真值定义在 service.CtxPermission（internal/service/datascope.go），
	// 这里做别名以避免两处常量漂移 —— DataScope 的读取方在 service 包内。
	CtxPermission = service.CtxPermission
)

// JWTAuth 等价于 JwtAuthenticationTokenFilter + TokenService.verifyToken：
//  1. 取 Authorization 头并剥离 "Bearer " 前缀（前端恒定带前缀）
//  2. 解析 JWT 拿到 uuid，再取 Redis 中的 login_tokens:<uuid>
//  3. 剩余有效期不足 20 分钟自动刷新
//
// 失败响应对齐 AuthenticationEntryPointImpl.java:30-32：code=401。
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := jwtx.StripBearer(c.GetHeader(config.C.Token.Header))
		if token == "" {
			unauthorized(c)
			return
		}
		uuidInToken, _, err := jwtx.Parse(token)
		if err != nil {
			unauthorized(c)
			return
		}
		loginUser, ok, err := service.GetLoginUser(uuidInToken)
		if err != nil || !ok || loginUser == nil {
			unauthorized(c)
			return
		}
		if err := service.VerifyToken(loginUser); err != nil {
			result.Fail(c, result.Error(err.Error()))
			c.Abort()
			return
		}
		c.Set(CtxLoginUser, loginUser)
		c.Next()
	}
}

// LoginUser 从 gin.Context 取登录态。
func LoginUser(c *gin.Context) (*entity.LoginUser, bool) {
	v, ok := c.Get(CtxLoginUser)
	if !ok {
		return nil, false
	}
	lu, ok := v.(*entity.LoginUser)
	return lu, ok && lu != nil
}

func unauthorized(c *gin.Context) {
	msg := fmt.Sprintf("请求访问：%s，认证失败，无法访问系统资源", c.Request.URL.Path)
	c.JSON(http.StatusOK, result.ErrorCode(config.StatusUnauthorized, msg))
	c.Abort()
}
