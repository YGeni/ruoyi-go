// Package handler 对应 ruoyi-admin 的 Controller 层。
package handler

import (
	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/middleware"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/service"
	"github.com/YGeni/ruoyi-go/server-go/pkg/captcha"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证相关接口。
type AuthHandler struct{}

// CaptchaImage GET /captchaImage
//
// 等价于 CaptchaController.java:45-93：返回 { captchaEnabled, uuid, img }，
// img 为**不含 data URI 前缀**的裸 base64（前端 src/views/login.vue:143-149 自行拼接）。
func (AuthHandler) CaptchaImage(c *gin.Context) {
	uuid, img, err := captcha.Generate()
	if err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}
	ajax := result.Success().
		Put("captchaEnabled", true).
		Put("uuid", uuid).
		Put("img", img)
	result.OK(c, ajax)
}

// Login POST /login
//
// 等价 SysLoginController.java:56-65：成功返回 { code:200, msg:"操作成功", token:"xxx" }。
func (AuthHandler) Login(c *gin.Context) {
	var body service.LoginBody
	if err := c.ShouldBindJSON(&body); err != nil {
		result.Fail(c, result.Error(config.MsgUserNotExists))
		return
	}
	// 第三个参数是 User-Agent：Java 的 TokenService.setUserAgent(L163-171)
	// 从 ServletUtils.getRequest() 里取，用于写 LoginUser 的 browser / os，
	// 最终显示在"在线用户"页面。
	token, err := service.Login(body, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}
	result.OK(c, result.Success().Put(config.Token, token))
}

// Register POST /register
//
// 等价 SysRegisterController.java:28-39：先看 sys.account.registerUser 开关，
// 关着就直接 error；开了才走 SysRegisterService.register 的校验链。
//
// 匿名放行（SecurityConfig.java 的 permitAll），因此**不挂** JWTAuth。
func (AuthHandler) Register(c *gin.Context) {
	var body service.RegisterBody
	if err := c.ShouldBindJSON(&body); err != nil {
		result.Fail(c, result.Error(config.MsgUserNotExists))
		return
	}
	if err := service.Register(body, c.ClientIP(), c.GetHeader("User-Agent")); err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// UnlockScreen POST /unlockscreen
//
// 等价 SysIndexController.java:43-63：校验当前登录用户的密码，成功回
// success("解锁成功")。
//
// 需要登录态（Java 的 SecurityUtils.getUsername() 依赖 SecurityContext），
// 但**不在** permitAll 列表里，所以挂 JWTAuth + 空权限串。
func (AuthHandler) UnlockScreen(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		result.Fail(c, result.Error(service.MsgUnlockPwdEmpty))
		return
	}
	if err := service.UnlockScreen(lu, body.Password); err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}
	result.OK(c, result.SuccessMsg(service.MsgUnlockSuccess))
}

// Logout POST /logout
//
// 等价 LogoutSuccessHandlerImpl.java:47-51：删除 login_tokens:<uuid> 登录态。
func (AuthHandler) Logout(c *gin.Context) {
	if err := service.Logout(c.GetHeader(config.C.Token.Header)); err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}
	result.OK(c, result.SuccessMsg(config.MsgLogoutSuccess))
}

// GetInfo GET /getInfo
//
// 等价 SysLoginController.java:72-94：返回 user / roles / permissions /
// pwdChrtype / isDefaultModifyPwd / isPasswordExpired 六个字段，
// 缺任一字段前端 src/store/modules/user.js:24-85 都会异常。
func (AuthHandler) GetInfo(c *gin.Context) {
	lu, ok := middleware.LoginUser(c)
	if !ok {
		result.Fail(c, result.ErrorCode(config.StatusUnauthorized, "认证失败，无法访问系统资源"))
		return
	}
	user := lu.User
	roles, err := service.GetRolePermission(user)
	if err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}
	permissions, err := service.GetMenuPermission(user)
	if err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}
	// 权限发生变化时刷新缓存（SysLoginController.java:81-85）
	if !samePerms(lu.Permissions, permissions) {
		lu.Permissions = permissions
		if err := service.RefreshToken(lu); err != nil {
			result.Fail(c, result.Error(err.Error()))
			return
		}
	}
	ajax := result.Success().
		Put("user", user).
		Put("roles", roles).
		Put("permissions", permissions).
		Put("pwdChrtype", service.AccountChrtype()).
		Put("isDefaultModifyPwd", service.InitPasswordIsModify(user.PwdUpdateDate)).
		Put("isPasswordExpired", service.PasswordIsExpiration(user.PwdUpdateDate))
	result.OK(c, ajax)
}

// GetRouters GET /getRouters
//
// 等价 SysLoginController.java:101-107：返回 { code:200, data:[RouterVo] }，
// 结构必须满足前端 src/store/modules/permission.js:59-97 的解析。
func (AuthHandler) GetRouters(c *gin.Context) {
	lu, ok := middleware.LoginUser(c)
	if !ok {
		result.Fail(c, result.ErrorCode(config.StatusUnauthorized, "认证失败，无法访问系统资源"))
		return
	}
	menus, err := service.SelectMenuTreeByUserID(lu.UserID)
	if err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}
	routers := service.BuildMenus(menus)
	if routers == nil {
		routers = []entity.RouterVo{}
	}
	result.OK(c, result.SuccessData(routers))
}

func samePerms(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]struct{}, len(a))
	for _, p := range a {
		set[p] = struct{}{}
	}
	for _, p := range b {
		if _, ok := set[p]; !ok {
			return false
		}
	}
	return true
}
