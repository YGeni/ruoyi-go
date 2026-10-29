// Package router 注册 HTTP 路由。
package router

import (
	"fmt"
	"net/http"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/handler"
	"github.com/YGeni/ruoyi-go/server-go/internal/handler/system"
	"github.com/YGeni/ruoyi-go/server-go/internal/middleware"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

// New 构建 gin 引擎。
//
// 端口固定 8080（config.Server.Port），与前端 vite.config.js:5 的代理目标一致；
// context-path 为 /（对齐 application.yml 的 server.servlet.context-path）。
func New() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// 对齐 application.yml: spring.servlet.multipart.max-request-size。
	//
	// gin 的 MaxMultipartMemory 只决定"多大以内走内存、超出落临时文件"，
	// **不会拒绝请求**；真正的硬闸在下面两处：
	//   1. r.MaxMultipartMemory：让大文件落盘而不是吃内存
	//   2. limitBody：读 body 时直接按 max-request-size 截断
	r.MaxMultipartMemory = config.C.Spring.Servlet.Multipart.MaxRequestSizeBytes()

	auth := handler.AuthHandler{}
	common := handler.CommonHandler{}

	// 静态资源：等价
	//
	//	registry.addResourceHandler(Constants.RESOURCE_PREFIX + "/**")
	//	        .addResourceLocations("file:" + RuoYiConfig.getProfile() + "/");
	//
	// sys_user.avatar 存的就是 "/profile/avatar/2026/10/28/xxx.png"，
	// 前端 userAvatar.vue:136 拼成 VITE_APP_BASE_API + imgUrl 后直接当 <img src>，
	// 没有这个映射头像预览会 404（前端不做容错，直接显示破图）。
	//
	// 免鉴权：对 GET /profile/** 是 permitAll。
	// http.FileServer 自身会清掉 ".."，不存在目录穿越。
	//
	// 注意 profile 目录可能还不存在（还没上传过任何文件），
	// gin.Static 对不存在的根目录不会 panic，只是请求返回 404，与 Java 行为一致。
	r.Static(config.ResourcePrefix, config.C.RuoYi.Profile)

	// 匿名放行：对齐的 /login、/captchaImage
	r.GET("/captchaImage", auth.CaptchaImage)
	r.POST("/login", auth.Login)
	r.POST("/register", auth.Register)

	// 需要登录态但无需权限串（不在 perm_table 内，直接注册）
	r.POST("/logout", middleware.JWTAuth(), auth.Logout)
	r.GET("/getInfo", middleware.JWTAuth(), auth.GetInfo)
	r.GET("/getRouters", middleware.JWTAuth(), auth.GetRouters)
	r.POST("/unlockscreen", middleware.JWTAuth(), auth.UnlockScreen)

	user := system.UserHandler{}
	menu := system.MenuHandler{}
	monitor := system.MonitorHandler{}
	role := system.RoleHandler{}
	dept := system.DeptHandler{}
	post := system.PostHandler{}
	dict := system.DictHandler{}
	cfg := system.ConfigHandler{}
	notice := system.NoticeHandler{}

	// ------------------------------------------------------------ 通用上传下载
	//
	// 四个端点在 permRoutes 里权限串均为空（对齐 Java：CommonController 无 @PreAuthorize，
	// SysProfileController 同样无），因此 perm() 只挂 JWTAuth。
	//
	// limitBody 必须套在上传接口上：超限要回 application/json 的错误体，
	// 前端 src/utils/ruoyi.js:225-228 才能把 Blob 还原成提示。
	perm(r, "POST", "/common/upload", limitBody(), common.Upload)
	perm(r, "POST", "/common/uploads", limitBody(), common.Uploads)
	perm(r, "GET", "/common/download/resource", common.ResourceDownload)
	perm(r, "GET", "/common/download", common.Download)

	profile := system.ProfileHandler{}
	// ------------------------------------------------------------ 系统管理 - 菜单
	perm(r, "GET", "/system/menu/list", menu.List)
	perm(r, "GET", "/system/menu/treeselect", menu.TreeSelect)
	perm(r, "GET", "/system/menu/roleMenuTreeselect/:roleId", menu.RoleMenuTreeSelect)
	perm(r, "POST", "/system/menu", menu.Add)
	perm(r, "PUT", "/system/menu", menu.Edit)
	perm(r, "PUT", "/system/menu/updateSort", menu.UpdateSort)
	// :menuId 通配段必须最后注册，否则会吃掉上面的静态段
	perm(r, "DELETE", "/system/menu/:menuId", menu.Remove)
	perm(r, "GET", "/system/menu/:menuId", menu.GetInfo)

	// ------------------------------------------------------------ 系统监控 - 在线用户
	// "数据监控"菜单 iframe 的是 ${VITE_APP_BASE_API}/druid/login.html
	// （src/views/monitor/druid/index.vue:11）。Java 侧这个地址由 druid 的
	// StatViewServlet 提供，Go 没有 druid，所以这里挂一个自建的对等页面。
	// 只校验登录态、不挂权限串 —— 与 /unlockscreen 一样直接注册：
	// Java 的 /druid/* 也不走 Shiro，是 druid 自己的登录页管的。
	r.GET("/druid/login.html", monitor.DruidPage)
	r.GET("/druid/index.html", monitor.DruidPage)
	// "系统接口"菜单 iframe 的是 ${VITE_APP_BASE_API}/swagger-ui/index.html
	// （src/views/tool/swagger/index.vue:11）。Java 侧由 springdoc-openapi 提供，
	// Go 侧在 openapi.go / swagger_page.go 自建：/v3/api-docs 出 OpenAPI 3.0，
	// /swagger-ui/index.html 出只读页面。同样不挂权限串。
	r.GET("/v3/api-docs", OpenAPIDoc)
	// /swagger-ui/* 一个通配路由包住：index.html（真 Swagger UI）、
	// progress.html（自绘迁移进度视图）、以及 css/js 静态资源。
	// 不能各注册一条 —— gin 不允许同一位置既有静态段又有通配段。
	r.GET("/swagger-ui/*filepath", SwaggerUIResource)
	// springdoc 在 application.yml 里配的 path（/swagger-ui.html）
	r.GET("/swagger-ui.html", SwaggerUIResource)

	perm(r, "GET", "/monitor/online/list", monitor.OnlineList)
	perm(r, "DELETE", "/monitor/online/:tokenId", monitor.ForceLogout)
	perm(r, "GET", "/monitor/logininfor/list", monitor.LogininforList)
	perm(r, "POST", "/monitor/logininfor/export", monitor.LogininforExport)
	perm(r, "DELETE", "/monitor/logininfor/:infoIds", monitor.LogininforRemove)
	perm(r, "DELETE", "/monitor/logininfor/clean", monitor.LogininforClean)
	perm(r, "GET", "/monitor/logininfor/unlock/:userName", monitor.LogininforUnlock)
	perm(r, "GET", "/monitor/operlog/list", monitor.OperLogList)
	perm(r, "POST", "/monitor/operlog/export", monitor.OperLogExport)
	perm(r, "DELETE", "/monitor/operlog/:operIds", monitor.OperLogRemove)
	perm(r, "DELETE", "/monitor/operlog/clean", monitor.OperLogClean)
	perm(r, "GET", "/monitor/cache", monitor.CacheInfo)
	perm(r, "GET", "/monitor/cache/getNames", monitor.CacheNames)
	perm(r, "GET", "/monitor/cache/getKeys/:cacheName", monitor.CacheKeys)
	perm(r, "GET", "/monitor/cache/getValue/:cacheName/:cacheKey", monitor.CacheValue)
	perm(r, "DELETE", "/monitor/cache/clearCacheName/:cacheName", monitor.ClearCacheName)
	perm(r, "DELETE", "/monitor/cache/clearCacheKey/:cacheKey", monitor.ClearCacheKey)
	perm(r, "DELETE", "/monitor/cache/clearCacheAll", monitor.ClearCacheAll)
	perm(r, "GET", "/monitor/server", monitor.ServerInfo)
	job := system.JobHandler{}
	perm(r, "GET", "/monitor/job/list", job.JobList)
	perm(r, "POST", "/monitor/job/export", job.JobExport)
	perm(r, "GET", "/monitor/job/:jobId", job.JobGet)
	perm(r, "POST", "/monitor/job", job.JobAdd)
	perm(r, "PUT", "/monitor/job", job.JobEdit)
	perm(r, "PUT", "/monitor/job/changeStatus", job.JobChangeStatus)
	perm(r, "PUT", "/monitor/job/run", job.JobRun)
	perm(r, "DELETE", "/monitor/job/:jobIds", job.JobRemove)
	perm(r, "GET", "/monitor/jobLog/list", job.JobLogList)
	perm(r, "POST", "/monitor/jobLog/export", job.JobLogExport)
	perm(r, "GET", "/monitor/jobLog/:jobLogId", job.JobLogGet)
	perm(r, "DELETE", "/monitor/jobLog/:jobLogIds", job.JobLogRemove)
	perm(r, "DELETE", "/monitor/jobLog/clean", job.JobLogClean)
	gen := system.GenHandler{}
	perm(r, "GET", "/tool/gen/list", gen.GenList)
	perm(r, "GET", "/tool/gen/db/list", gen.GenDbList)
	perm(r, "GET", "/tool/gen/:tableId", gen.GenGet)
	perm(r, "GET", "/tool/gen/preview/:tableId", gen.GenPreview)
	perm(r, "GET", "/tool/gen/genCode/:tableName", gen.GenCode)
	perm(r, "GET", "/tool/gen/synchDb/:tableName", gen.GenSynchDb)
	perm(r, "POST", "/tool/gen/importTable", gen.GenImport)
	perm(r, "POST", "/tool/gen/createTable", gen.GenCreateTable)
	perm(r, "PUT", "/tool/gen", gen.GenEdit)
	perm(r, "DELETE", "/tool/gen/:tableIds", gen.GenRemove)
	perm(r, "GET", "/tool/gen/column/:tableId", gen.GenColumnList)
	perm(r, "GET", "/tool/gen/download/:tableName", gen.GenDownload)
	perm(r, "GET", "/tool/gen/batchGenCode", gen.GenBatchGenCode)

	// ------------------------------------------------------------ 个人信息
	//
	// /system/user/profile 必须注册在 /system/user/:userId **之前**，
	// 否则 gin 会把 "profile" 当 userId 解析（虽然 gin 支持静态/通配共存，
	// 但显式在前更不容易被后续改动破坏）。
	perm(r, "GET", "/system/user/profile", profile.GetProfile)
	perm(r, "PUT", "/system/user/profile", profile.UpdateProfile)
	perm(r, "POST", "/system/user/profile/avatar", limitBody(), profile.Avatar)
	perm(r, "PUT", "/system/user/profile/updatePwd", profile.UpdatePwd)

	// ------------------------------------------------------------ 系统管理 - 用户
	//
	// perm 的权限串全部从 perm_table.go 的 permRoutes 查（见 perm 的说明），
	// 路由即文档：忘了登记 -> 启动直接 panic，不会静默"裸奔"。
	//
	// 静态段必须注册在通配段之前（可读性；gin 本身支持两者共存）。
	perm(r, "GET", "/system/user/list", user.List)
	perm(r, "GET", "/system/user/deptTree", user.DeptTree)
	perm(r, "GET", "/system/user/authRole/:userId", user.AuthRole)
	perm(r, "PUT", "/system/user/authRole", user.InsertAuthRole)
	perm(r, "PUT", "/system/user/resetPwd", user.ResetPwd)
	perm(r, "PUT", "/system/user/changeStatus", user.ChangeStatus)
	perm(r, "GET", "/system/user/", user.GetInfo)
	perm(r, "GET", "/system/user/:userId", user.GetInfo)
	perm(r, "POST", "/system/user", user.Add)
	perm(r, "PUT", "/system/user", user.Edit)
	perm(r, "DELETE", "/system/user/:userIds", user.Remove)

	// ------------------------------------------------------------ 系统管理 - 角色
	perm(r, "GET", "/system/role/list", role.List)
	perm(r, "GET", "/system/role/optionselect", role.OptionSelect)
	perm(r, "GET", "/system/role/deptTree/:roleId", role.DeptTree)
	perm(r, "GET", "/system/role/authUser/allocatedList", role.AllocatedList)
	perm(r, "GET", "/system/role/authUser/unallocatedList", role.UnallocatedList)
	perm(r, "PUT", "/system/role/authUser/cancel", role.AuthUserCancel)
	perm(r, "PUT", "/system/role/authUser/cancelAll", role.AuthUserCancelAll)
	perm(r, "PUT", "/system/role/authUser/selectAll", role.AuthUserSelectAll)
	perm(r, "PUT", "/system/role/dataScope", role.DataScope)
	perm(r, "PUT", "/system/role/changeStatus", role.ChangeStatus)
	perm(r, "GET", "/system/role/:roleId", role.GetInfo)
	perm(r, "POST", "/system/role", role.Add)
	perm(r, "PUT", "/system/role", role.Edit)
	perm(r, "DELETE", "/system/role/:roleIds", role.Remove)

	// ------------------------------------------------------------ 系统管理 - 部门
	perm(r, "GET", "/system/dept/list", dept.List)
	perm(r, "GET", "/system/dept/list/exclude/:deptId", dept.ListExclude)
	perm(r, "PUT", "/system/dept/updateSort", dept.UpdateSort)
	perm(r, "GET", "/system/dept/:deptId", dept.GetInfo)
	perm(r, "POST", "/system/dept", dept.Add)
	perm(r, "PUT", "/system/dept", dept.Edit)
	perm(r, "DELETE", "/system/dept/:deptId", dept.Remove)

	// ------------------------------------------------------------ 系统管理 - 岗位
	perm(r, "GET", "/system/post/list", post.List)
	perm(r, "GET", "/system/post/optionselect", post.OptionSelect)
	perm(r, "GET", "/system/post/:postId", post.GetInfo)
	perm(r, "POST", "/system/post", post.Add)
	perm(r, "PUT", "/system/post", post.Edit)
	perm(r, "DELETE", "/system/post/:postIds", post.Remove)

	// ------------------------------------------------------------ 系统管理 - 字典
	perm(r, "GET", "/system/dict/type/list", dict.TypeList)
	perm(r, "GET", "/system/dict/type/optionselect", dict.TypeOptionSelect)
	perm(r, "DELETE", "/system/dict/type/refreshCache", dict.TypeRefreshCache)
	perm(r, "GET", "/system/dict/type/:dictId", dict.TypeGetInfo)
	perm(r, "POST", "/system/dict/type", dict.TypeAdd)
	perm(r, "PUT", "/system/dict/type", dict.TypeEdit)
	perm(r, "DELETE", "/system/dict/type/:dictIds", dict.TypeRemove)

	perm(r, "GET", "/system/dict/data/list", dict.DataList)
	perm(r, "GET", "/system/dict/data/type/:dictType", dict.DataByType)
	perm(r, "GET", "/system/dict/data/:dictCode", dict.DataGetInfo)
	perm(r, "POST", "/system/dict/data", dict.DataAdd)
	perm(r, "PUT", "/system/dict/data", dict.DataEdit)
	perm(r, "DELETE", "/system/dict/data/:dictCodes", dict.DataRemove)

	// ------------------------------------------------------------ 系统管理 - 参数
	perm(r, "GET", "/system/config/list", cfg.List)
	perm(r, "GET", "/system/config/configKey/:configKey", cfg.GetByKey)
	perm(r, "DELETE", "/system/config/refreshCache", cfg.RefreshCache)
	perm(r, "GET", "/system/config/:configId", cfg.GetInfo)
	perm(r, "POST", "/system/config", cfg.Add)
	perm(r, "PUT", "/system/config", cfg.Edit)
	perm(r, "DELETE", "/system/config/:configIds", cfg.Remove)

	// ------------------------------------------------------------ 系统管理 - 通知公告
	//
	// 后 4 个是 3.9 新增端点（listTop / markRead / markReadAll / readUsers/list），
	// 必须注册在 /system/notice/:noticeId 之前。
	perm(r, "GET", "/system/notice/list", notice.List)
	perm(r, "GET", "/system/notice/listTop", notice.ListTop)
	perm(r, "POST", "/system/notice/markRead", notice.MarkRead)
	perm(r, "POST", "/system/notice/markReadAll", notice.MarkReadAll)
	perm(r, "GET", "/system/notice/readUsers/list", notice.ReadUsers)
	perm(r, "GET", "/system/notice/:noticeId", notice.GetInfo)
	perm(r, "POST", "/system/notice", notice.Add)
	perm(r, "PUT", "/system/notice", notice.Edit)
	perm(r, "DELETE", "/system/notice/:noticeIds", notice.Remove)

	// 抓一份已注册路由的快照，供 /v3/api-docs 与"系统接口"页面标注
	// 「已实现 / 二期延后」（见 openapi.go）。
	snapshotRoutes(r)

	return r
}

// perm 注册一条需要登录态的路由，权限串**从 perm_table.go 的 permRoutes 查**。
//
// 这样权限串只有一份事实来源（perm_table.go），由
// internal/router/perm_diff_test.go 与 Java 注解做双向 diff；
// 路由注册处不再重复写字面量，漏登记会在启动时 panic 而不是静默放行。
//
// Permi 必须排在 JWTAuth 之后：它从 context 取 LoginUser，并把权限串写入
// context 供 service.DataScope 读取（等价 Java 的 PermissionContextHolder）。
// limitBody 按 multipart.max-request-size 限制请求体大小。
//
// 超限回 AjaxResult（code=500 + 中文文案）而不是裸 413/500 文本：
// 前端下载/上传用的是 responseType: 'blob'，必须靠 Content-Type=application/json
// 才能把响应还原成错误提示（src/utils/ruoyi.js:225-228）。
func limitBody() gin.HandlerFunc {
	max := config.C.Spring.Servlet.Multipart.MaxRequestSizeBytes()
	return func(c *gin.Context) {
		if c.Request.Body != nil && c.Request.ContentLength > max {
			result.Fail(c, result.Error(fmt.Sprintf(config.MsgUploadExceedRequestSize, max/1024/1024)))
			c.Abort()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)
		c.Next()
	}
}

func perm(r *gin.Engine, method, path string, h gin.HandlerFunc, extra ...gin.HandlerFunc) {
	p, ok := PermOf(method, path)
	if !ok {
		panic(fmt.Sprintf("路由 %s %s 未在 perm_table.go 的 permRoutes 中登记，"+
			"请先补登记（或确认它属于二期延后、不应注册）", method, path))
	}
	handlers := []gin.HandlerFunc{middleware.JWTAuth()}
	handlers = append(handlers, extra...)
	if p != "" {
		handlers = append(handlers, middleware.Permi(p))
	}
	handlers = append(handlers, h)
	r.Handle(method, path, handlers...)
}
