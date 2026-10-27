// Package router 注册 HTTP 路由。
package router

import (
	"github.com/YGeni/ruoyi-go/server-go/internal/handler"
	"github.com/YGeni/ruoyi-go/server-go/internal/middleware"

	"github.com/gin-gonic/gin"
)

// New 构建 gin 引擎。
//
// 端口固定 8080（config.Server.Port），与前端 vite.config.js:5 的代理目标一致；
// context-path 为 /（对齐 application.yml 的 server.servlet.context-path）。
func New() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	auth := handler.AuthHandler{}

	// 匿名放行：对齐 SecurityConfig.java:101-108 的 /login、/captchaImage
	r.GET("/captchaImage", auth.CaptchaImage)
	r.POST("/login", auth.Login)

	// 需要登录态
	r.POST("/logout", middleware.JWTAuth(), auth.Logout)
	r.GET("/getInfo", middleware.JWTAuth(), auth.GetInfo)
	r.GET("/getRouters", middleware.JWTAuth(), auth.GetRouters)

	return r
}
