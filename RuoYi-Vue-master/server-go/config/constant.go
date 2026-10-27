package config

// 本文件常量全部照搬 Java 侧常量类，禁止自定义取值，保证 Go 与 Java 两端行为一致。
//
// 来源：
//   - CacheConstants: ruoyi-common/src/main/java/com/ruoyi/common/constant/CacheConstants.java
//   - Constants:      ruoyi-common/src/main/java/com/ruoyi/common/constant/Constants.java
//   - UserConstants:  ruoyi-common/src/main/java/com/ruoyi/common/constant/UserConstants.java
//   - HttpStatus:     ruoyi-common/src/main/java/com/ruoyi/common/constant/HttpStatus.java

// ---------- CacheConstants：Redis key 前缀（与 Java 完全一致，两端共用同一份缓存）----------
const (
	LoginTokenKey   = "login_tokens:"  // CacheConstants.java:13
	CaptchaCodeKey  = "captcha_codes:" // CacheConstants.java:18
	RepeatSubmitKey = "repeat_submit:" // CacheConstants.java:33
	RateLimitKey    = "rate_limit:"    // CacheConstants.java:38
	PwdErrCntKey    = "pwd_err_cnt:"   // CacheConstants.java:43
	SysDictKey      = "sys_dict:"      // CacheConstants.java:20
	SysConfigKey    = "sys_config:"    // CacheConstants.java
)

// ---------- Constants ----------
const (
	Token        = "token"          // Constants.java:101 登录响应里 token 字段名
	TokenPrefix  = "Bearer "        // Constants.java:106 含尾空格
	LoginUserKey = "login_user_key" // Constants.java:111 JWT claim：存放 uuid
	JWTUsername  = "sub"            // Constants.java:121 == Claims.SUBJECT
	JWTUserID    = "userid"         // Constants.java:116

	AllPermission       = "*:*:*" // Constants.java:76  权限直通
	SuperAdmin          = "admin" // Constants.java      角色直通
	PermissionDelimiter = ","     // Constants.java:86
	RoleDelimiter       = ","     // Constants.java:91

	CaptchaExpiration = 2 // Constants.java:96 验证码有效期（分钟）

	// MILLIS_MINUTE_TWENTY = 20*60*1000，TokenService.java:53
	// 令牌剩余有效期不足 20 分钟即自动刷新
	MillisMinuteTwenty = 20 * 60 * 1000

	HTTP  = "http://"
	HTTPS = "https://"
	WWW   = "www."
)

// ---------- UserConstants ----------
const (
	// 菜单类型（char(1)）
	TypeDir    = "M" // UserConstants.java:49
	TypeMenu   = "C" // UserConstants.java:52
	TypeButton = "F" // UserConstants.java:55

	// 是否外链（char(1)，0 是外链 1 否）
	YesFrame = "0" // UserConstants.java:43
	NoFrame  = "1" // UserConstants.java:46

	// 组件标识（前端 permission.js:59-83 依赖这三个字面量）
	Layout     = "Layout"     // UserConstants.java:58
	ParentView = "ParentView" // UserConstants.java:61
	InnerLink  = "InnerLink"  // UserConstants.java:64

	// 状态（char(1)）
	Normal  = "0" // UserConstants.java:16
	Disable = "1" // UserConstants.java:22
)

// ---------- HttpStatus（前端 request.js:76-109 依赖这些 code）----------
const (
	StatusSuccess      = 200 // HttpStatus.java:13
	StatusUnauthorized = 401 // HttpStatus.java:53
	StatusForbidden    = 403 // HttpStatus.java:58
	StatusError        = 500 // HttpStatus.java:83
	StatusWarn         = 601 // HttpStatus.java:93
)

// ---------- 业务提示文案（i18n/messages.properties）----------
const (
	MsgUserNotExists       = "用户不存在/密码错误" // user.not.exists / user.password.not.match
	MsgUserDeleted         = "对不起，您的账号已被删除"
	MsgUserBlocked         = "用户已封禁，请联系管理员"
	MsgCaptchaExpire       = "验证码已失效"
	MsgCaptchaError        = "验证码错误"
	MsgLoginBlocked        = "很遗憾，访问IP已被列入系统黑名单"
	MsgLoginSuccess        = "登录成功"
	MsgLogoutSuccess       = "退出成功"
	MsgOperateSuccess      = "操作成功"
	MsgOperateFailed       = "操作失败"
	MsgQuerySuccess        = "查询成功"
	MsgNoPermission        = "没有权限，请联系管理员授权" // GlobalExceptionHandler.java:35-41
	MsgPasswordRetryExceed = "密码输入错误%d次，帐户锁定%d分钟"
)

// ---------- sys_config 配置键 ----------
const (
	ConfigCaptchaEnabled     = "sys.account.captchaEnabled"
	ConfigChrtype            = "sys.account.chrtype"
	ConfigInitPasswordModify = "sys.account.initPasswordModify"
	ConfigPasswordValidDays  = "sys.account.passwordValidateDays"
)
