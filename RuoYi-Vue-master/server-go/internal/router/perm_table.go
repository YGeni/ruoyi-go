package router

// 权限串常量表：替代 Java 侧 @PreAuthorize("@ss.hasPermi('...')") 注解。
//
// 取值全部来自 sys_menu.perms，与 Java 注解字面量一致。
// 本文件（常量 + permRoutes 表）由 internal/router/perm_diff_test.go 与 Java 控制器源码
// 做**双向 diff** 守护：Java 侧增删注解、或本表写错字面量，测试即失败（可作 CI 断言）。
//
// 全量：Java 控制器共 148 个端点、其中 116 个带权限注解、去重后 71 个权限串。
// 1b 落地范围（permRoutes）覆盖 system/{user,role,dept,post,dict,config,notice,profile}
// 与 common/*，共 75 个端点；其余（monitor/*、tool/*、system:menu:*、Excel 导入导出）
// 属二期，只登记常量不绑定路由，由 diff 测试归入「二期延后」清单。

// Go 常量名生成规则：Perm + 各段首字母大写（system:user:resetPwd -> PermSystemUserResetPwd）。
const (
	// monitor:cache
	PermMonitorCacheList = "monitor:cache:list"
	// monitor:job
	PermMonitorJobAdd          = "monitor:job:add"
	PermMonitorJobChangeStatus = "monitor:job:changeStatus"
	PermMonitorJobEdit         = "monitor:job:edit"
	PermMonitorJobExport       = "monitor:job:export"
	PermMonitorJobList         = "monitor:job:list"
	PermMonitorJobQuery        = "monitor:job:query"
	PermMonitorJobRemove       = "monitor:job:remove"
	// monitor:logininfor
	PermMonitorLogininforExport = "monitor:logininfor:export"
	PermMonitorLogininforList   = "monitor:logininfor:list"
	PermMonitorLogininforRemove = "monitor:logininfor:remove"
	PermMonitorLogininforUnlock = "monitor:logininfor:unlock"
	// monitor:online
	PermMonitorOnlineForceLogout = "monitor:online:forceLogout"
	PermMonitorOnlineList        = "monitor:online:list"
	// monitor:operlog
	PermMonitorOperlogExport = "monitor:operlog:export"
	PermMonitorOperlogList   = "monitor:operlog:list"
	PermMonitorOperlogRemove = "monitor:operlog:remove"
	// monitor:server
	PermMonitorServerList = "monitor:server:list"
	// system:config
	PermSystemConfigAdd    = "system:config:add"
	PermSystemConfigEdit   = "system:config:edit"
	PermSystemConfigExport = "system:config:export"
	PermSystemConfigList   = "system:config:list"
	PermSystemConfigQuery  = "system:config:query"
	PermSystemConfigRemove = "system:config:remove"
	// system:dept
	PermSystemDeptAdd    = "system:dept:add"
	PermSystemDeptEdit   = "system:dept:edit"
	PermSystemDeptList   = "system:dept:list"
	PermSystemDeptQuery  = "system:dept:query"
	PermSystemDeptRemove = "system:dept:remove"
	// system:dict
	PermSystemDictAdd    = "system:dict:add"
	PermSystemDictEdit   = "system:dict:edit"
	PermSystemDictExport = "system:dict:export"
	PermSystemDictList   = "system:dict:list"
	PermSystemDictQuery  = "system:dict:query"
	PermSystemDictRemove = "system:dict:remove"
	// system:menu
	PermSystemMenuAdd    = "system:menu:add"
	PermSystemMenuEdit   = "system:menu:edit"
	PermSystemMenuList   = "system:menu:list"
	PermSystemMenuQuery  = "system:menu:query"
	PermSystemMenuRemove = "system:menu:remove"
	// system:notice
	PermSystemNoticeAdd    = "system:notice:add"
	PermSystemNoticeEdit   = "system:notice:edit"
	PermSystemNoticeList   = "system:notice:list"
	PermSystemNoticeRemove = "system:notice:remove"
	// system:post
	PermSystemPostAdd    = "system:post:add"
	PermSystemPostEdit   = "system:post:edit"
	PermSystemPostExport = "system:post:export"
	PermSystemPostList   = "system:post:list"
	PermSystemPostQuery  = "system:post:query"
	PermSystemPostRemove = "system:post:remove"
	// system:role
	PermSystemRoleAdd    = "system:role:add"
	PermSystemRoleEdit   = "system:role:edit"
	PermSystemRoleExport = "system:role:export"
	PermSystemRoleList   = "system:role:list"
	PermSystemRoleQuery  = "system:role:query"
	PermSystemRoleRemove = "system:role:remove"
	// system:user
	PermSystemUserAdd      = "system:user:add"
	PermSystemUserEdit     = "system:user:edit"
	PermSystemUserExport   = "system:user:export"
	PermSystemUserImport   = "system:user:import"
	PermSystemUserList     = "system:user:list"
	PermSystemUserQuery    = "system:user:query"
	PermSystemUserRemove   = "system:user:remove"
	PermSystemUserResetPwd = "system:user:resetPwd"
	// tool:gen
	PermToolGenCode    = "tool:gen:code"
	PermToolGenEdit    = "tool:gen:edit"
	PermToolGenImport  = "tool:gen:import"
	PermToolGenList    = "tool:gen:list"
	PermToolGenPreview = "tool:gen:preview"
	PermToolGenQuery   = "tool:gen:query"
	PermToolGenRemove  = "tool:gen:remove"
)

// RoutePerm 一条「路由 - 权限」绑定，与 Java 控制器的方法级注解一一对应。
type RoutePerm struct {
	Method string // GET/POST/PUT/DELETE
	Path   string // gin 风格路径（Java 的 {id} 记作 :id）
	Perm   string // 权限串；为空表示仅需登录态，不加 Permi 中间件
	Java   string // 来源，便于 diff 定位
}

// permRoutes 1b 范围内的全量绑定表（75 条）。
//
// router.go 注册路由时通过 PermOf(method, path) 取权限串并挂 middleware.Permi。
var permRoutes = []RoutePerm{
	{"GET", "/common/download", "", "CommonController"},
	{"GET", "/common/download/resource", "", "CommonController"},
	{"POST", "/common/upload", "", "CommonController"},
	{"POST", "/common/uploads", "", "CommonController"},
	{"POST", "/system/config", PermSystemConfigAdd, "SysConfigController"},
	{"PUT", "/system/config", PermSystemConfigEdit, "SysConfigController"},
	{"GET", "/system/config/configKey/:configKey", "", "SysConfigController"},
	{"GET", "/system/config/list", PermSystemConfigList, "SysConfigController"},
	{"DELETE", "/system/config/refreshCache", PermSystemConfigRemove, "SysConfigController"},
	{"DELETE", "/system/config/:configIds", PermSystemConfigRemove, "SysConfigController"},
	{"GET", "/system/config/:configId", PermSystemConfigQuery, "SysConfigController"},
	{"POST", "/system/dept", PermSystemDeptAdd, "SysDeptController"},
	{"PUT", "/system/dept", PermSystemDeptEdit, "SysDeptController"},
	{"GET", "/system/dept/list", PermSystemDeptList, "SysDeptController"},
	{"GET", "/system/dept/list/exclude/:deptId", PermSystemDeptList, "SysDeptController"},
	{"PUT", "/system/dept/updateSort", PermSystemDeptEdit, "SysDeptController"},
	{"DELETE", "/system/dept/:deptId", PermSystemDeptRemove, "SysDeptController"},
	{"GET", "/system/dept/:deptId", PermSystemDeptQuery, "SysDeptController"},
	{"POST", "/system/dict/data", PermSystemDictAdd, "SysDictDataController"},
	{"PUT", "/system/dict/data", PermSystemDictEdit, "SysDictDataController"},
	{"GET", "/system/dict/data/list", PermSystemDictList, "SysDictDataController"},
	{"GET", "/system/dict/data/type/:dictType", "", "SysDictDataController"},
	{"DELETE", "/system/dict/data/:dictCodes", PermSystemDictRemove, "SysDictDataController"},
	{"GET", "/system/dict/data/:dictCode", PermSystemDictQuery, "SysDictDataController"},
	{"POST", "/system/dict/type", PermSystemDictAdd, "SysDictTypeController"},
	{"PUT", "/system/dict/type", PermSystemDictEdit, "SysDictTypeController"},
	{"GET", "/system/dict/type/list", PermSystemDictList, "SysDictTypeController"},
	{"GET", "/system/dict/type/optionselect", "", "SysDictTypeController"},
	{"DELETE", "/system/dict/type/refreshCache", PermSystemDictRemove, "SysDictTypeController"},
	{"DELETE", "/system/dict/type/:dictIds", PermSystemDictRemove, "SysDictTypeController"},
	{"GET", "/system/dict/type/:dictId", PermSystemDictQuery, "SysDictTypeController"},
	{"POST", "/system/notice", PermSystemNoticeAdd, "SysNoticeController"},
	{"PUT", "/system/notice", PermSystemNoticeEdit, "SysNoticeController"},
	{"GET", "/system/notice/list", PermSystemNoticeList, "SysNoticeController"},
	// SysMenuController.java
	{"POST", "/system/menu", PermSystemMenuAdd, "SysMenuController"},
	{"PUT", "/system/menu", PermSystemMenuEdit, "SysMenuController"},
	{"GET", "/system/menu/list", PermSystemMenuList, "SysMenuController"},
	{"GET", "/system/menu/roleMenuTreeselect/:roleId", "", "SysMenuController"},
	{"GET", "/system/menu/treeselect", "", "SysMenuController"},
	{"PUT", "/system/menu/updateSort", PermSystemMenuEdit, "SysMenuController"},
	{"DELETE", "/system/menu/:menuId", PermSystemMenuRemove, "SysMenuController"},
	{"GET", "/system/menu/:menuId", PermSystemMenuQuery, "SysMenuController"},
	// CacheController.java
	{"GET", "/monitor/cache", PermMonitorCacheList, "CacheController"},
	{"GET", "/monitor/cache/getNames", PermMonitorCacheList, "CacheController"},
	{"GET", "/monitor/cache/getKeys/:cacheName", PermMonitorCacheList, "CacheController"},
	{"GET", "/monitor/cache/getValue/:cacheName/:cacheKey", PermMonitorCacheList, "CacheController"},
	{"DELETE", "/monitor/cache/clearCacheName/:cacheName", PermMonitorCacheList, "CacheController"},
	{"DELETE", "/monitor/cache/clearCacheKey/:cacheKey", PermMonitorCacheList, "CacheController"},
	{"DELETE", "/monitor/cache/clearCacheAll", PermMonitorCacheList, "CacheController"},
	// ServerController.java
	{"GET", "/monitor/server", PermMonitorServerList, "ServerController"},
	// SysJobController.java
	{"GET", "/monitor/job/list", PermMonitorJobList, "SysJobController"},
	{"POST", "/monitor/job/export", PermMonitorJobExport, "SysJobController"},
	{"GET", "/monitor/job/:jobId", PermMonitorJobQuery, "SysJobController"},
	{"POST", "/monitor/job", PermMonitorJobAdd, "SysJobController"},
	{"PUT", "/monitor/job", PermMonitorJobEdit, "SysJobController"},
	{"PUT", "/monitor/job/changeStatus", PermMonitorJobChangeStatus, "SysJobController"},
	{"PUT", "/monitor/job/run", PermMonitorJobChangeStatus, "SysJobController"},
	{"DELETE", "/monitor/job/:jobIds", PermMonitorJobRemove, "SysJobController"},
	// SysJobLogController.java
	{"GET", "/monitor/jobLog/list", PermMonitorJobList, "SysJobLogController"},
	{"POST", "/monitor/jobLog/export", PermMonitorJobExport, "SysJobLogController"},
	{"GET", "/monitor/jobLog/:jobLogId", PermMonitorJobQuery, "SysJobLogController"},
	{"DELETE", "/monitor/jobLog/:jobLogIds", PermMonitorJobRemove, "SysJobLogController"},
	{"DELETE", "/monitor/jobLog/clean", PermMonitorJobRemove, "SysJobLogController"},
	// SysIndexController.java（/unlockscreen 需登录态但无权限串，见 router.go 直接注册）
	{"POST", "/unlockscreen", "", "SysIndexController"},
	// SysRegisterController.java（匿名放行，见 router.go 直接注册）
	{"POST", "/register", "", "SysRegisterController"},
	// SysUserOnlineController.java
	{"GET", "/monitor/online/list", PermMonitorOnlineList, "SysUserOnlineController"},
	{"DELETE", "/monitor/online/:tokenId", PermMonitorOnlineForceLogout, "SysUserOnlineController"},
	// SysLogininforController.java
	{"GET", "/monitor/logininfor/list", PermMonitorLogininforList, "SysLogininforController"},
	{"POST", "/monitor/logininfor/export", PermMonitorLogininforExport, "SysLogininforController"},
	{"DELETE", "/monitor/logininfor/:infoIds", PermMonitorLogininforRemove, "SysLogininforController"},
	{"DELETE", "/monitor/logininfor/clean", PermMonitorLogininforRemove, "SysLogininforController"},
	{"GET", "/monitor/logininfor/unlock/:userName", PermMonitorLogininforUnlock, "SysLogininforController"},
	// SysOperlogController.java
	{"GET", "/monitor/operlog/list", PermMonitorOperlogList, "SysOperlogController"},
	{"POST", "/monitor/operlog/export", PermMonitorOperlogExport, "SysOperlogController"},
	{"DELETE", "/monitor/operlog/:operIds", PermMonitorOperlogRemove, "SysOperlogController"},
	{"DELETE", "/monitor/operlog/clean", PermMonitorOperlogRemove, "SysOperlogController"},
	{"GET", "/system/notice/listTop", "", "SysNoticeController"},
	{"POST", "/system/notice/markRead", "", "SysNoticeController"},
	{"POST", "/system/notice/markReadAll", "", "SysNoticeController"},
	{"GET", "/system/notice/readUsers/list", PermSystemNoticeList, "SysNoticeController"},
	{"DELETE", "/system/notice/:noticeIds", PermSystemNoticeRemove, "SysNoticeController"},
	{"GET", "/system/notice/:noticeId", "", "SysNoticeController"},
	{"POST", "/system/post", PermSystemPostAdd, "SysPostController"},
	{"PUT", "/system/post", PermSystemPostEdit, "SysPostController"},
	{"GET", "/system/post/list", PermSystemPostList, "SysPostController"},
	{"GET", "/system/post/optionselect", "", "SysPostController"},
	{"DELETE", "/system/post/:postIds", PermSystemPostRemove, "SysPostController"},
	{"GET", "/system/post/:postId", PermSystemPostQuery, "SysPostController"},
	{"POST", "/system/role", PermSystemRoleAdd, "SysRoleController"},
	{"PUT", "/system/role", PermSystemRoleEdit, "SysRoleController"},
	{"GET", "/system/role/authUser/allocatedList", PermSystemRoleList, "SysRoleController"},
	{"PUT", "/system/role/authUser/cancel", PermSystemRoleEdit, "SysRoleController"},
	{"PUT", "/system/role/authUser/cancelAll", PermSystemRoleEdit, "SysRoleController"},
	{"PUT", "/system/role/authUser/selectAll", PermSystemRoleEdit, "SysRoleController"},
	{"GET", "/system/role/authUser/unallocatedList", PermSystemRoleList, "SysRoleController"},
	{"PUT", "/system/role/changeStatus", PermSystemRoleEdit, "SysRoleController"},
	{"PUT", "/system/role/dataScope", PermSystemRoleEdit, "SysRoleController"},
	{"GET", "/system/role/deptTree/:roleId", PermSystemRoleQuery, "SysRoleController"},
	{"GET", "/system/role/list", PermSystemRoleList, "SysRoleController"},
	{"GET", "/system/role/optionselect", PermSystemRoleQuery, "SysRoleController"},
	{"DELETE", "/system/role/:roleIds", PermSystemRoleRemove, "SysRoleController"},
	{"GET", "/system/role/:roleId", PermSystemRoleQuery, "SysRoleController"},
	{"POST", "/system/user", PermSystemUserAdd, "SysUserController"},
	{"PUT", "/system/user", PermSystemUserEdit, "SysUserController"},
	{"GET", "/system/user/", PermSystemUserQuery, "SysUserController"},
	{"PUT", "/system/user/authRole", PermSystemUserEdit, "SysUserController"},
	{"GET", "/system/user/authRole/:userId", PermSystemUserQuery, "SysUserController"},
	{"PUT", "/system/user/changeStatus", PermSystemUserEdit, "SysUserController"},
	{"GET", "/system/user/deptTree", PermSystemUserList, "SysUserController"},
	{"GET", "/system/user/list", PermSystemUserList, "SysUserController"},
	{"GET", "/system/user/profile", "", "SysProfileController"},
	{"PUT", "/system/user/profile", "", "SysProfileController"},
	{"POST", "/system/user/profile/avatar", "", "SysProfileController"},
	{"PUT", "/system/user/profile/updatePwd", "", "SysProfileController"},
	{"PUT", "/system/user/resetPwd", PermSystemUserResetPwd, "SysUserController"},
	{"DELETE", "/system/user/:userIds", PermSystemUserRemove, "SysUserController"},
	{"GET", "/system/user/:userId", PermSystemUserQuery, "SysUserController"},
}

// permRoutesPhase2 二期延后端点清单（30 条）。
//
// **本表不参与路由绑定**，仅作为交接清单存在：
//   - 唯一事实来源是 internal/router/perm_diff_test.go 的 scopeFiles /
//     deferPathKeywords —— 二期每落地一个 handler，把对应控制器文件加进 scopeFiles，
//     该控制器下的端点就会从「二期延后」自动转为「范围内」，diff 测试随即要求绑定；
//   - TestPhase2ListMatchesJava 守护本表与 Java 源码一致，改 Java 不改这里会红；
//   - 未翻入 permRoutes 的条目在 diff 测试里计入「二期延后」，不判失败，
//     即当前是"不实现也绿"，交接时必须逐条 grep -n "TODO(phase2)" 过一遍。
var permRoutesPhase2 = []RoutePerm{
	// CaptchaController.java
	{"GET", "/captchaImage", "", "CaptchaController"}, // TODO(phase2) CaptchaController.java:31
	// GenController.java
	{"DELETE", "/tool/gen/:tableIds", PermToolGenRemove, "GenController"},      // TODO(phase2) GenController.java:178
	{"GET", "/tool/gen/:tableId", PermToolGenQuery, "GenController"},           // TODO(phase2) GenController.java:70
	{"GET", "/tool/gen/batchGenCode", PermToolGenCode, "GenController"},        // TODO(phase2) GenController.java:241
	{"GET", "/tool/gen/column/:tableId", PermToolGenList, "GenController"},     // TODO(phase2) GenController.java:99
	{"GET", "/tool/gen/db/list", PermToolGenList, "GenController"},             // TODO(phase2) GenController.java:87
	{"GET", "/tool/gen/download/:tableName", PermToolGenCode, "GenController"}, // TODO(phase2) GenController.java:201
	{"GET", "/tool/gen/genCode/:tableName", PermToolGenCode, "GenController"},  // TODO(phase2) GenController.java:213
	{"GET", "/tool/gen/list", PermToolGenList, "GenController"},                // TODO(phase2) GenController.java:49
	{"GET", "/tool/gen/preview/:tableId", PermToolGenPreview, "GenController"}, // TODO(phase2) GenController.java:190
	{"GET", "/tool/gen/synchDb/:tableName", PermToolGenEdit, "GenController"},  // TODO(phase2) GenController.java:229
	{"POST", "/tool/gen/createTable", "", "GenController"},                     // TODO(phase2) GenController.java:128
	{"POST", "/tool/gen/importTable", PermToolGenImport, "GenController"},      // TODO(phase2) GenController.java:113
	{"PUT", "/tool/gen", PermToolGenEdit, "GenController"},                     // TODO(phase2) GenController.java:165
	// ServerController.java
	// SysConfigController.java
	{"POST", "/system/config/export", PermSystemConfigExport, "SysConfigController"}, // TODO(phase2) SysConfigController.java:49
	// SysDictDataController.java
	{"POST", "/system/dict/data/export", PermSystemDictExport, "SysDictDataController"}, // TODO(phase2) SysDictDataController.java:52
	// SysDictTypeController.java
	{"POST", "/system/dict/type/export", PermSystemDictExport, "SysDictTypeController"}, // TODO(phase2) SysDictTypeController.java:46
	// SysIndexController.java
	// SysLoginController.java
	{"GET", "getInfo", "", "SysLoginController"},    // TODO(phase2) SysLoginController.java:72
	{"GET", "getRouters", "", "SysLoginController"}, // TODO(phase2) SysLoginController.java:101
	{"POST", "/login", "", "SysLoginController"},    // TODO(phase2) SysLoginController.java:35
	// SysLogininforController.java
	// SysOperlogController.java
	// SysPostController.java
	{"POST", "/system/post/export", PermSystemPostExport, "SysPostController"}, // TODO(phase2) SysPostController.java:49
	// SysRegisterController.java
	// SysRoleController.java
	{"POST", "/system/role/export", PermSystemRoleExport, "SysRoleController"}, // TODO(phase2) SysRoleController.java:65
	// SysUserController.java
	{"POST", "/system/user/export", PermSystemUserExport, "SysUserController"},     // TODO(phase2) SysUserController.java:68
	{"POST", "/system/user/importData", PermSystemUserImport, "SysUserController"}, // TODO(phase2) SysUserController.java:78
	{"POST", "/system/user/importTemplate", "", "SysUserController"},               // TODO(phase2) SysUserController.java:90
	// SysUserOnlineController.java
	// TestController.java
	{"DELETE", "/test/user/:userId", "", "TestController"}, // TODO(phase2) TestController.java:73
	{"GET", "/test/user/:userId", "", "TestController"},    // TODO(phase2) TestController.java:46
	{"GET", "/test/user/list", "", "TestController"},       // TODO(phase2) TestController.java:38
	{"POST", "/test/user/save", "", "TestController"},      // TODO(phase2) TestController.java:61
	{"PUT", "/test/user/update", "", "TestController"},     // TODO(phase2) TestController.java:73
}

// PermOf 按 method + gin 路径查登记的权限串。
// 返回 ("", false) 表示未登记；("", true) 表示已登记但无需权限校验。
func PermOf(method, path string) (string, bool) {
	for _, r := range permRoutes {
		if r.Method == method && r.Path == path {
			return r.Perm, true
		}
	}
	return "", false
}

// permConstants 全部权限串（71 个），供 diff 测试校验"Java 权限串必须有对应常量"。
var permConstants = []string{
	PermMonitorCacheList,
	PermMonitorJobAdd,
	PermMonitorJobChangeStatus,
	PermMonitorJobEdit,
	PermMonitorJobExport,
	PermMonitorJobList,
	PermMonitorJobQuery,
	PermMonitorJobRemove,
	PermMonitorLogininforExport,
	PermMonitorLogininforList,
	PermMonitorLogininforRemove,
	PermMonitorLogininforUnlock,
	PermMonitorOnlineForceLogout,
	PermMonitorOnlineList,
	PermMonitorOperlogExport,
	PermMonitorOperlogList,
	PermMonitorOperlogRemove,
	PermMonitorServerList,
	PermSystemConfigAdd,
	PermSystemConfigEdit,
	PermSystemConfigExport,
	PermSystemConfigList,
	PermSystemConfigQuery,
	PermSystemConfigRemove,
	PermSystemDeptAdd,
	PermSystemDeptEdit,
	PermSystemDeptList,
	PermSystemDeptQuery,
	PermSystemDeptRemove,
	PermSystemDictAdd,
	PermSystemDictEdit,
	PermSystemDictExport,
	PermSystemDictList,
	PermSystemDictQuery,
	PermSystemDictRemove,
	PermSystemMenuAdd,
	PermSystemMenuEdit,
	PermSystemMenuList,
	PermSystemMenuQuery,
	PermSystemMenuRemove,
	PermSystemNoticeAdd,
	PermSystemNoticeEdit,
	PermSystemNoticeList,
	PermSystemNoticeRemove,
	PermSystemPostAdd,
	PermSystemPostEdit,
	PermSystemPostExport,
	PermSystemPostList,
	PermSystemPostQuery,
	PermSystemPostRemove,
	PermSystemRoleAdd,
	PermSystemRoleEdit,
	PermSystemRoleExport,
	PermSystemRoleList,
	PermSystemRoleQuery,
	PermSystemRoleRemove,
	PermSystemUserAdd,
	PermSystemUserEdit,
	PermSystemUserExport,
	PermSystemUserImport,
	PermSystemUserList,
	PermSystemUserQuery,
	PermSystemUserRemove,
	PermSystemUserResetPwd,
	PermToolGenCode,
	PermToolGenEdit,
	PermToolGenImport,
	PermToolGenList,
	PermToolGenPreview,
	PermToolGenQuery,
	PermToolGenRemove,
}
