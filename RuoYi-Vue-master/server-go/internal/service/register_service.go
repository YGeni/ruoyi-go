package service

import (
	"errors"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/captcha"
	"github.com/YGeni/ruoyi-go/server-go/pkg/securityx"
)

// 注册接口的文案，对齐 SysRegisterService.java 与 SysIndexController.java 的原句。
const (
	MsgRegisterDisabled  = "当前系统没有开启注册功能！"
	MsgUsernameEmpty     = "用户名不能为空"
	MsgPasswordEmpty     = "用户密码不能为空"
	MsgUsernameLength    = "账户长度必须在2到20个字符之间"
	MsgPasswordLength    = "密码长度必须在5到20个字符之间"
	MsgRegisterSuccess   = "注册成功"
	MsgUnlockSuccess     = "解锁成功"
	MsgUnlockPasswordErr = "密码错误，请重新输入"
	MsgUnlockTimeout     = "服务器超时，请重新登录"
	MsgUnlockPwdEmpty    = "密码不能为空"
)

// RegisterBody 注册入参，等价 com.ruoyi.common.core.domain.model.RegisterBody
// （继承 SysUser，额外带 code / uuid）。
type RegisterBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Code     string `json:"code"`
	UUID     string `json:"uuid"`
}

// Register 注册，等价 SysRegisterService.register(L38-82)。
//
// 返回非 nil error 时，handler 用其文案回 error(msg)（Java 的
// `StringUtils.isEmpty(msg) ? success() : error(msg)`）。
//
// DIFF(review): Java 注册成功后异步写一条 sys_logininfor，status 取
// Constants.SUCCESS（AsyncFactory 里 REGISTER 与 LOGIN_SUCCESS 一样归为成功，
// '0'），msg 是 user.register.success。Go 侧同步写同一条，字段一致。
func Register(body RegisterBody, ip, userAgent string) error {
	on, err := registerEnabled()
	if err != nil {
		return err
	}
	if !on {
		return errors.New(MsgRegisterDisabled)
	}
	// 验证码开关（SysRegisterService.java:46-50）
	if enabled, err := captchaEnabled(); err != nil {
		return err
	} else if enabled {
		if body.UUID == "" || !captcha.Verify(body.UUID, body.Code) {
			return errors.New(config.MsgCaptchaExpire)
		}
	}
	switch {
	case body.Username == "":
		return errors.New(MsgUsernameEmpty)
	case body.Password == "":
		return errors.New(MsgPasswordEmpty)
	case len(body.Username) < 2 || len(body.Username) > 20:
		return errors.New(MsgUsernameLength)
	case len(body.Password) < 5 || len(body.Password) > 20:
		return errors.New(MsgPasswordLength)
	}
	unique, err := repository.CheckUserNameUnique(body.Username, 0)
	if err != nil {
		return err
	}
	if !unique {
		return errors.New("保存用户'" + body.Username + "'失败，注册账号已存在")
	}
	hashed, err := securityx.Encrypt(body.Password)
	if err != nil {
		return err
	}
	now := time.Now()
	if err := InsertUser(&entity.SysUser{
		UserName:      body.Username,
		NickName:      body.Username, // SysRegisterService.java:75 setNickName(username)
		Password:      hashed,
		PwdUpdateDate: &now,          // SysRegisterService.java:76
		Status:        config.Normal, // '0' 正常
		DelFlag:       "0",
	}, nil); err != nil {
		return err
	}
	recordLogin(body.Username, ip, userAgent, LoginStatusSuccess, MsgRegisterSuccess)
	return nil
}

// registerEnabled 读 sys.account.registerUser 开关。
func registerEnabled() (bool, error) {
	v, _, err := repository.SelectConfigValueByKey(config.ConfigRegisterUser)
	if err != nil {
		return false, err
	}
	return v == "true", nil
}

// UnlockScreen 解锁屏幕，等价 SysIndexController.unlockScreen(L43-63)。
//
// 只校验密码，不下发新 token：前端锁屏是纯前端状态（store 里存了 password），
// token 一直有效，解锁只是把锁屏遮罩去掉。
func UnlockScreen(lu *entity.LoginUser, password string) error {
	if password == "" {
		return errors.New(MsgUnlockPwdEmpty)
	}
	if lu == nil || lu.User == nil {
		return errors.New(MsgUnlockTimeout)
	}
	// 以库里的为准：LoginUser 快照里的 User.Password 是空（entity 上 json:"-"），
	// 必须回查一次，否则永远匹配不上。
	u, err := repository.SelectUserByUserName(lu.Username())
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New(MsgUnlockTimeout)
	}
	if err := securityx.Validate(u.UserName, password, u.Password); err != nil {
		if errors.Is(err, securityx.ErrPasswordNotMatch) || errors.Is(err, securityx.ErrPasswordRetryExceed) {
			return errors.New(MsgUnlockPasswordErr)
		}
		return err
	}
	return nil
}
