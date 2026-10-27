package router

// 权限串常量表：替代 Java 侧 115 处 @PreAuthorize("@ss.hasPermi('...')") 注解。
//
// 取值全部来自 sys_menu.perms，与 Java 注解字面量一致
// （实测：@ss.hasPermi 共 115 处，分布见 AGENTS.md §10.a）。
//
// 本期（1a）只登记常量，接口尚未绑定权限校验；
// 1b 逐条在路由上调用 router.Permi(PermSystemUserList) 完成绑定，
// 并用脚本与 Java 注解清单做双向 diff（AGENTS.md §12 风险 1）。

const (
	// 系统管理 - 用户
	PermSystemUserList   = "system:user:list"
	PermSystemUserQuery  = "system:user:query"
	PermSystemUserAdd    = "system:user:add"
	PermSystemUserEdit   = "system:user:edit"
	PermSystemUserRemove = "system:user:remove"
	PermSystemUserReset  = "system:user:resetPwd"

	// 系统管理 - 角色
	PermSystemRoleList   = "system:role:list"
	PermSystemRoleQuery  = "system:role:query"
	PermSystemRoleAdd    = "system:role:add"
	PermSystemRoleEdit   = "system:role:edit"
	PermSystemRoleRemove = "system:role:remove"

	// 系统管理 - 部门 / 岗位 / 菜单
	PermSystemDeptList = "system:dept:list"
	PermSystemPostList = "system:post:list"
	PermSystemMenuList = "system:menu:list"

	// 系统管理 - 字典 / 参数 / 通知
	PermSystemDictList   = "system:dict:list"
	PermSystemConfigList = "system:config:list"
	PermSystemNoticeList = "system:notice:list"

	// 系统监控
	PermMonitorOperlogList    = "monitor:operlog:list"
	PermMonitorLogininforList = "monitor:logininfor:list"
	PermMonitorOnlineList     = "monitor:online:list"
	PermMonitorJobList        = "monitor:job:list"
	PermMonitorCacheList      = "monitor:cache:list"

	// 系统工具
	PermToolGenList = "tool:gen:list"
)
