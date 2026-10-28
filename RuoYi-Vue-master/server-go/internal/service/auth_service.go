// Package service 对应 Java 侧的 Service 层。
package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/captcha"
	"github.com/YGeni/ruoyi-go/server-go/pkg/jwtx"
	"github.com/YGeni/ruoyi-go/server-go/pkg/redisx"
	"github.com/YGeni/ruoyi-go/server-go/pkg/securityx"
	"github.com/YGeni/ruoyi-go/server-go/pkg/uax"

	"github.com/google/uuid"
)

// 登录状态（sys_logininfor.status，char(1)：'0'成功 '1'失败）
const (
	LoginStatusSuccess = "0"
	LoginStatusFail    = "1"
)

// LoginBody 登录入参，等价于 com.ruoyi.common.core.domain.model.LoginBody。
// 字段名与前端 src/views/login.vue / src/api/login.js:4-18 保持一致。
type LoginBody struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Code     string `json:"code"`
	UUID     string `json:"uuid"`
}

// Login 登录主流程，等价于 SysLoginService.login(L63-100)：
// 验证码校验 -> 前置校验 -> 用户状态校验 -> 密码校验 -> 记录登录信息 -> 签发令牌。
//
// 返回 JWT 字符串（前端存入 Cookie: Admin-Token，请求头带 "Bearer " 前缀）。
func Login(body LoginBody, ip, userAgent string) (string, error) {
	if err := validateCaptcha(body, ip, userAgent); err != nil {
		return "", err
	}
	if err := loginPreCheck(body, ip, userAgent); err != nil {
		return "", err
	}
	user, err := loadUserByUsername(body.Username, ip)
	if err != nil {
		return "", err
	}
	// 密码校验 + 失败计数（pwd_err_cnt:<username>，5 次锁 10 分钟）
	if err := securityx.Validate(body.Username, body.Password, user.Password); err != nil {
		recordLogin(body.Username, ip, userAgent, LoginStatusFail, config.MsgUserNotExists)
		if errors.Is(err, securityx.ErrPasswordNotMatch) || errors.Is(err, securityx.ErrPasswordRetryExceed) {
			return "", err
		}
		return "", err
	}
	// 记录登录日志并更新最后登录信息
	recordLogin(body.Username, ip, userAgent, LoginStatusSuccess, config.MsgLoginSuccess)
	if err := repository.UpdateLoginInfo(user.UserID, ip, time.Now()); err != nil {
		return "", err
	}
	return createToken(user, ip, userAgent)
}

// Logout 退出登录，等价于 LogoutSuccessHandlerImpl.java:47-51：删除 Redis 登录态。
func Logout(token string) error {
	u := jwtx.StripBearer(token)
	if u == "" {
		return nil
	}
	uuidInToken, _, err := jwtx.Parse(u)
	if err != nil {
		return nil
	}
	return redisx.Del(config.LoginTokenKey + uuidInToken)
}

// validateCaptcha 等价于 SysLoginService.validateCaptcha(L110-129)。
func validateCaptcha(body LoginBody, ip, userAgent string) error {
	enabled, err := captchaEnabled()
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	if body.UUID == "" || !captcha.Verify(body.UUID, body.Code) {
		recordLogin(body.Username, ip, userAgent, LoginStatusFail, config.MsgCaptchaExpire)
		return errors.New(config.MsgCaptchaExpire)
	}
	return nil
}

// loginPreCheck 等价于 SysLoginService.loginPreCheck(L136-165)。
func loginPreCheck(body LoginBody, ip, userAgent string) error {
	// 用户名 2-20 位、密码 5-20 位（UserConstants:73-80）
	if len(body.Username) < 2 || len(body.Username) > 20 || len(body.Password) < 5 || len(body.Password) > 20 {
		recordLogin(body.Username, ip, userAgent, LoginStatusFail, config.MsgUserNotExists)
		return errors.New(config.MsgUserNotExists)
	}
	// IP 黑名单（sys.login.blackIPList）
	if black, ok, _ := repository.SelectConfigValueByKey("sys.login.blackIPList"); ok && black != "" {
		for _, item := range strings.Split(black, "|") {
			if strings.Contains(strings.TrimSpace(item), "*") {
				if strings.HasPrefix(ip, strings.TrimSuffix(strings.TrimSpace(item), "*")) {
					recordLogin(body.Username, ip, userAgent, LoginStatusFail, config.MsgLoginBlocked)
					return errors.New(config.MsgLoginBlocked)
				}
			} else if strings.TrimSpace(item) == ip {
				recordLogin(body.Username, ip, userAgent, LoginStatusFail, config.MsgLoginBlocked)
				return errors.New(config.MsgLoginBlocked)
			}
		}
	}
	return nil
}

// loadUserByUsername 等价于 UserDetailsServiceImpl.loadUserByUsername(L37-60)。
func loadUserByUsername(username, ip string) (*entity.SysUser, error) {
	user, err := repository.SelectUserByUserName(username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		recordLogin(username, ip, "", LoginStatusFail, config.MsgUserNotExists)
		return nil, errors.New(config.MsgUserNotExists)
	}
	if model.IsDeleted(user.DelFlag) {
		recordLogin(username, ip, "", LoginStatusFail, config.MsgUserDeleted)
		return nil, errors.New(config.MsgUserDeleted)
	}
	if user.Status == config.Disable {
		recordLogin(username, ip, "", LoginStatusFail, config.MsgUserBlocked)
		return nil, errors.New(config.MsgUserBlocked)
	}
	return user, nil
}

// ---------- 令牌：对应 TokenService.java ----------

// createToken 等价于 TokenService.createToken(L115-126)。
func createToken(user *entity.SysUser, ip, userAgent string) (string, error) {
	perms, err := GetMenuPermission(user)
	if err != nil {
		return "", err
	}
	token := uuid.NewString()
	loginUser := &entity.LoginUser{
		UserID:      user.UserID,
		DeptID:      user.DeptID,
		Token:       token,
		Ipaddr:      ip,
		Permissions: perms,
		User:        user,
	}
	// 等价 TokenService.setUserAgent(L163-171)：解析浏览器、操作系统与归属地。
	//
	// 这三个字段**只在登录时写入** Redis 快照，
	// "系统监控 > 在线用户"页面（SysUserOnlineServiceImpl.loginUserToUserOnline L76-94）
	// 直接读它们渲染 tokenId / ipaddr / loginLocation / browser / os / loginTime 六列。
	// 之前留空导致在线用户页这几列全空。
	loginUser.Browser = uax.Browser(userAgent)
	loginUser.Os = uax.OS(userAgent)
	loginUser.LoginLocation = uax.Location(ip)
	if err := RefreshToken(loginUser); err != nil {
		return "", err
	}
	return jwtx.Generate(token, user.UserName)
}

// RefreshToken 等价于 TokenService.refreshToken(L149-156)：
// 重置 loginTime / expireTime 并把 LoginUser 写进 login_tokens:<uuid>。
func RefreshToken(loginUser *entity.LoginUser) error {
	loginUser.LoginTime = time.Now().UnixMilli()
	loginUser.ExpireTime = loginUser.LoginTime +
		int64(config.C.Token.ExpireTime)*60*1000
	return redisx.SetObject(config.LoginTokenKey+loginUser.Token, loginUser,
		config.C.Token.TokenExpireDuration())
}

// GetLoginUser 取登录态，等价于 TokenService.getLoginUser(L63-84)。
func GetLoginUser(uuidInToken string) (*entity.LoginUser, bool, error) {
	var lu entity.LoginUser
	ok, err := redisx.GetObject(config.LoginTokenKey+uuidInToken, &lu)
	if err != nil {
		return nil, false, err
	}
	return &lu, ok, nil
}

// VerifyToken 等价于 TokenService.verifyToken(L134-142)：
// 剩余有效期不足 20 分钟时自动刷新缓存。
func VerifyToken(loginUser *entity.LoginUser) error {
	if loginUser.ExpireTime-time.Now().UnixMilli() <= config.MillisMinuteTwenty {
		return RefreshToken(loginUser)
	}
	return nil
}

// ---------- 辅助 ----------

func captchaEnabled() (bool, error) {
	v, ok, err := repository.SelectConfigValueByKey(config.ConfigCaptchaEnabled)
	if err != nil {
		return false, err
	}
	if !ok {
		// 无该配置项时默认开启，与 SysConfigServiceImpl.selectCaptchaEnabled 的默认行为一致
		return true, nil
	}
	return strings.EqualFold(strings.TrimSpace(v), "true"), nil
}

// recordLogin 同步写入 sys_logininfor，等价 AsyncFactory.recordLogininfor
// （AsyncFactory.java:73-90）：用户名、IP、归属地、浏览器、操作系统、状态、消息、时间。
//
// 这五个字段就是"系统监控 > 登录日志"页面渲染的列，
// 与登录态快照（createToken 里写的那一份）取的是同一套 uax 解析结果。
func recordLogin(username, ip, userAgent, status, msg string) {
	_ = repository.InsertLogininfor(&entity.SysLogininfor{
		UserName:      username,
		Ipaddr:        ip,
		LoginLocation: uax.Location(ip),
		Browser:       uax.Browser(userAgent),
		Os:            uax.OS(userAgent),
		Status:        status,
		Msg:           msg,
		LoginTime:     time.Now(),
	})
}

// ErrBusiness 业务错误（用于向前端返回 500 + 具体文案）。
type ErrBusiness struct{ Msg string }

func (e *ErrBusiness) Error() string { return e.Msg }

// NewBusinessError 构造业务错误。
func NewBusinessError(format string, args ...any) error {
	return &ErrBusiness{Msg: fmt.Sprintf(format, args...)}
}
