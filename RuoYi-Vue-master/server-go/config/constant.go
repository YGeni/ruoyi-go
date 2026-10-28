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

// ---------- Constants：文件路径前缀 ----------
const (
	// ResourcePrefix 资源访问前缀，Constants.java:141。
	//
	// sys_user.avatar 存的就是 "/profile/avatar/2026/10/28/xxx.jpg"
	// 这种带前缀的相对路径；下载时用 StripPrefix 去掉它再拼 profile 根目录。
	ResourcePrefix = "/profile"
)

// ---------- 文件扩展名白名单（MimeTypeUtils.java）----------
var (
	// ImageExtension 头像上传白名单，MimeTypeUtils.java:20。
	ImageExtension = []string{"bmp", "gif", "jpg", "jpeg", "png"}

	// DefaultAllowedExtension 通用上传白名单，MimeTypeUtils.java:29-39。
	//
	// 同时被 checkAllowDownload 复用（FileUtils.java:162）：
	// 不在该白名单内的扩展名一律拒绝下载。
	DefaultAllowedExtension = []string{
		// 图片
		"bmp", "gif", "jpg", "jpeg", "png",
		// word excel powerpoint
		"doc", "docx", "xls", "xlsx", "ppt", "pptx", "txt",
		// 压缩文件
		"rar", "zip", "gz", "bz2",
		// 视频格式
		"mp4", "avi", "rmvb",
		// pdf
		"pdf",
	}

	// FlashExtension / MediaExtension / VideoExtension 仅登记，
	// 通用上传与下载校验都不使用（Java 侧同样只在特定场景用）。
	FlashExtension = []string{"swf", "flv"}
	MediaExtension = []string{"swf", "flv", "mp3", "wav", "wma", "wmv", "mid", "avi", "mpg", "asf", "rm", "rmvb"}
	VideoExtension = []string{"mp4", "avi", "rmvb"}
)

// ---------- FileUploadUtils 常量 ----------
const (
	// DefaultMaxSize 单文件默认上限 50M，FileUploadUtils.java:29。
	//
	// DIFF(review): 这是 **FileUploadUtils 自己**的硬上限；application.yml 的
	// spring.servlet.multipart.max-file-size=10MB 是 Spring MVC 层的另一道闸。
	// 两者都在，Go 侧同样实现两道（multipart 配置见 config.yaml）。
	DefaultMaxSize = 50 * 1024 * 1024

	// DefaultFileNameLength 文件名最大长度 100，FileUploadUtils.java:34。
	DefaultFileNameLength = 100
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

	// 文件上传相关（i18n/messages.properties:29-30）
	MsgUploadExceedMaxSize = "上传的文件大小超出限制的文件大小！<br/>允许的文件最大大小是：%dMB！" // upload.exceed.maxSize
	// MsgUploadExceedRequestSize Spring 的 max-request-size 超限文案。
	//
	// DIFF(review): Java 侧这条由 Spring 直接抛 MaxUploadSizeExceededException，
	// GlobalExceptionHandler **没有**对应分支，返回体不是 AjaxResult；
	// 前端 src/utils/ruoyi.js:225-228 依赖 application/json 才能把 Blob 还原成提示，
	// 故 Go 侧统一补成 AjaxResult 风格（code=500 + 中文文案）。
	MsgUploadExceedRequestSize  = "上传的文件总大小超出限制！<br/>允许的文件总大小是：%dMB！"
	MsgUploadFilenameExceedLen  = "上传的文件名最长%d个字符"           // upload.filename.exceed.length
	MsgInvalidExtensionFmt      = "文件[%s]后缀[%s]不正确，请上传%s格式" // InvalidExtensionException.java:23
	MsgAllowDownloadFmt         = "文件名称(%s)非法，不允许下载。 "      // CommonController.java:52
	MsgResourceAllowDownloadFmt = "资源文件(%s)非法，不允许下载。 "      // CommonController.java:145
)

// ---------- sys_config 配置键 ----------
const (
	ConfigCaptchaEnabled     = "sys.account.captchaEnabled"
	ConfigRegisterUser       = "sys.account.registerUser"
	ConfigChrtype            = "sys.account.chrtype"
	ConfigInitPasswordModify = "sys.account.initPasswordModify"
	ConfigPasswordValidDays  = "sys.account.passwordValidateDays"
)
