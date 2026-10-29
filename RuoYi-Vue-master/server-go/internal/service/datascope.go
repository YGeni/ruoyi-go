// 本文件实现 @DataScope 数据权限过滤。
//
// 放在 internal/service 而非 internal/middleware 的原因：它**不是 HTTP 中间件**，
// 而是 GORM 的 Scope 构造器（func(*gorm.DB) *gorm.DB），挂在 gin 中间件链上是错的。
//
// 复刻依据（逐行对齐
//   - L50  admin 直通：currentUser.isAdmin() 时不拼任何条件
//   - L63  permission 取注解值，为空时回退 PermissionContextHolder.getContext()
//   - L67  scopeCustomIds：先扫一遍所有 CUSTOM 且 status=正常 且命中 permission 的角色
//   - L77  遍历角色；conditions.contains(dataScope) 去重；status=停用 跳过
//   - L81  permission 非空且角色不持有该权限 -> 跳过
//   - L92-94   "1" 全部：清空已有条件并 break
//   - L101-105 "2" 自定义：role_id = X（单个）或 role_id in (...)（多个）
//   - L110     "3" 本部门：d.dept_id = user.deptId
//   - L114     "4" 本部门及以下：d.dept_id IN (SELECT dept_id FROM sys_dept WHERE dept_id=? or find_in_set(?, ancestors))
//   - L120/125 "5" 仅本人：有 userAlias 时 u.user_id = userId，否则 d.dept_id = 0
//   - L132-135 兜底：conditions 为空（没有任何角色命中）时 OR d.dept_id = 0  —— 不查任何数据
//   - L143     最终注入 " AND (" + 去掉首个 " OR " 的条件串 + ")"
//
// 与 Java 的关键差异：**全部条件参数化**，不做任何 SQL 字符串拼接
// （AGENTS.md 全局规则 4；Java 侧是 ${params.dataScope} 直接拼串）。
package service

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/pkg/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 数据范围取值，与逐字一致（char(1)，一律 string）。
const (
	DataScopeAll       = "1" // 全部数据权限
	DataScopeCustom    = "2" // 自定数据权限
	DataScopeDept      = "3" // 部门数据权限
	DataScopeDeptChild = "4" // 部门及以下数据权限
	DataScopeSelf      = "5" // 仅本人数据权限
)

// CtxPermission 当前接口所需的权限串在 gin.Context 中的键。
//
// 等价于 Java 的 request-scoped PermissionContextHolder
// 读取 / :70 由 @PreAuthorize 写入）
// 由 middleware.Permi 写入、本文件的 DataScope 读取。
const CtxPermission = "permission"

// 表别名与字段名，等价于 @DataScope 的注解默认值/32/36）。
//
// 五个落点均未覆盖默认值（SysUserServiceImpl:76/89/102 用 deptAlias="d" userAlias="u"；
// SysRoleServiceImpl:54 与 SysDeptServiceImpl:45 用 deptAlias="d"），故字段名恒为这两个。
const (
	dataScopeDeptField = "dept_id"
	dataScopeUserField = "user_id"
)

// aliasPattern 表别名白名单。
//
// DIFF(review): Java 侧把 deptAlias/userAlias 直接 format 进 SQL 字符串；
// Go 侧先做白名单校验，非法别名降级为不加前缀，避免拼接出畸形 SQL。
var aliasPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ScopeInfo 描述本次数据权限过滤的命中情况。
//
// 存在的意义：把 Java 里"静默"的三条路径显式化，便于单测断言与线上排查：
//   - admin 直通（Applied=false）
//   - 命中"全部"（Applied=false，无过滤）
//   - **兜底 dept_id=0**（Fallback=true，结果必为 0 行）—— 这条最危险，
//     配置错误（例如角色 permissions 没装载）会静默把列表清空，
//     因此本文件在 Fallback 时打 WARN 并由调用方/单测显式断言。
type ScopeInfo struct {
	Applied  bool     // 是否施加了过滤条件
	Fallback bool     // 是否命中"空条件兜底 dept_id=0"（必然 0 行）
	Reason   string   // 命中原因，便于日志与单测断言
	Branches []string // 命中的数据范围分支（"1"/"2"/"3"/"4"/"5"）
	// RolePermMissed 记录因"角色未持有 permission"而被跳过的角色 key，
	// 用于识别"SysRole.Permissions 未装载"这类配置缺口（见 entity/sys_misc.go）。
	RolePermMissed []string
}

// DataScope 返回等价 @DataScope(deptAlias, userAlias) 的 GORM Scope。
//
// permission 从 gin.Context 的 CtxPermission 取（等价 PermissionContextHolder），
// 取值缺失时按 Java 语义传空串（空串表示"不按权限过滤角色"）。
//
// 用法：
//
//	rows, total, err := repository.SelectUserList(q, pg, service.DataScope(c, lu, "d", "u"))
func DataScope(c *gin.Context, u *entity.LoginUser, deptAlias, userAlias string) func(*gorm.DB) *gorm.DB {
	permission := ""
	if c != nil {
		permission = c.GetString(CtxPermission)
	}
	scope, info := DataScopeWithInfo(permission, u, deptAlias, userAlias)
	if info.Fallback {
		// 兜底意味着"该账号什么都看不到"，多半是角色/权限配置缺口，必须留痕。
		// logger.Sugar 在测试环境（未 Init）下为 nil，故判空。
		if logger.Sugar != nil {
			logger.Sugar.Warnf("DataScope 命中空条件兜底（%s.dept_id = 0）：userID=%d permission=%q reason=%s",
				deptAlias, userIDOf(u), permission, info.Reason)
		}
	}
	return scope
}

// DataScopeWithInfo 在 DataScope 之外额外返回命中信息，供单测断言"是否静默兜底"。
func DataScopeWithInfo(permission string, u *entity.LoginUser, deptAlias, userAlias string) (func(*gorm.DB) *gorm.DB, ScopeInfo) {
	// L50：admin 直通，不过滤任何数据
	if u == nil || u.User == nil || u.User.IsAdmin() {
		return noopScope(), ScopeInfo{Reason: "admin 直通，不过滤数据"}
	}
	user := u.User

	// 别名校验（见 aliasPattern 的 DIFF 说明）
	if deptAlias != "" && !aliasPattern.MatchString(deptAlias) {
		deptAlias = ""
	}
	if userAlias != "" && !aliasPattern.MatchString(userAlias) {
		userAlias = ""
	}
	deptCol := column(deptAlias, dataScopeDeptField)
	userCol := column(userAlias, dataScopeUserField)
	perms := splitPermission(permission)

	// L67：先收集所有 CUSTOM 且可用且命中 permission 的角色 ID
	var scopeCustomIDs []int64
	for i := range user.Roles {
		r := &user.Roles[i]
		if r.DataScope == DataScopeCustom && r.Status == config.Normal && rolePermOK(r, permission, perms) {
			scopeCustomIDs = append(scopeCustomIDs, r.RoleID)
		}
	}

	type cond struct {
		sql  string
		args []any
	}
	var (
		conds    []cond
		seen     = map[string]bool{} // 等价 Java 的 conditions 列表（按 dataScope 去重）
		missed   []string
		all      bool
		branches []string
	)

	for i := range user.Roles {
		r := &user.Roles[i]
		ds := r.DataScope
		// L77：同一 dataScope 只处理一次；停用角色跳过
		if seen[ds] || r.Status == config.Disable {
			continue
		}
		// L81：permission 非空但角色不持有 -> 跳过，并记录供排查
		if !rolePermOK(r, permission, perms) {
			missed = append(missed, fmt.Sprintf("%s(%s)", r.RoleKey, ds))
			continue
		}
		switch ds {
		case DataScopeAll:
			// L92-94：清空已有条件并跳出
			conds = nil
			branches = nil
			all = true
		case DataScopeCustom:
			if len(scopeCustomIDs) > 1 {
				conds = append(conds, cond{
					sql:  deptCol + " IN (SELECT dept_id FROM sys_role_dept WHERE role_id IN (?))",
					args: []any{scopeCustomIDs},
				})
			} else {
				conds = append(conds, cond{
					sql:  deptCol + " IN (SELECT dept_id FROM sys_role_dept WHERE role_id = ?)",
					args: []any{r.RoleID},
				})
			}
		case DataScopeDept:
			conds = append(conds, cond{sql: deptCol + " = ?", args: []any{deptIDOf(user)}})
		case DataScopeDeptChild:
			// find_in_set 参数化（Java 侧是 find_in_set({deptId}, ancestors) 拼串）
			conds = append(conds, cond{
				sql:  deptCol + " IN (SELECT dept_id FROM sys_dept WHERE dept_id = ? OR find_in_set(?, ancestors))",
				args: []any{deptIDOf(user), deptIDOf(user)},
			})
		case DataScopeSelf:
			if userAlias != "" {
				conds = append(conds, cond{sql: userCol + " = ?", args: []any{user.UserID}})
			} else {
				// L125：仅本人且没有 userAlias 别名 -> 不查询任何数据
				conds = append(conds, cond{sql: deptCol + " = 0", args: nil})
			}
		default:
			// 未知 data_scope：按 Java 语义不产生条件，但仍加入 conditions 参与去重，
			// 避免后续同值角色重复处理。
			seen[ds] = true
			continue
		}
		branches = append(branches, ds)
		seen[ds] = true
		if all {
			break
		}
	}

	if all {
		return noopScope(), ScopeInfo{Reason: "命中数据范围 1（全部），不过滤", Branches: []string{DataScopeAll}}
	}

	// L132-135：没有任何角色命中 -> 兜底不查任何数据
	if len(conds) == 0 {
		reason := "无角色命中数据范围"
		if len(missed) > 0 {
			reason = fmt.Sprintf("所有角色均未持有权限 %q（被跳过：%s）", permission, strings.Join(missed, ","))
		} else if len(user.Roles) == 0 {
			reason = "账号无任何角色"
		}
		return denyScope(deptCol), ScopeInfo{
			Applied:        true,
			Fallback:       true,
			Reason:         reason,
			RolePermMissed: missed,
		}
	}

	condsCopy := conds
	// 分组条件**必须**显式用 clause.OrConditions 构造，等价于 Java 的
	// " AND (" + 条件串 + ")"。
	//
	// DIFF(review): 试过两种更"自然"的写法，都被 gorm v1.25.12 坑了——
	//   1. tx.Where(func(d *gorm.DB) *gorm.DB{...})：Statement.BuildCondition
	//      （statement.go:285-341）**没有 func 分支**，落到 default 被当成
	//      "按主键等值"，实测生成 `u`.`user_id` = <func 指针> 这类错误 SQL。
	//   2. 嵌套 *gorm.DB（tx.Where(sub)）+ sub.Where(..).Or(..)：
	//      最终走 clause.Or(clause.And(..)) 路径，而 AndConditions.Build
	//      （clause/where.go:121-129）只在 len(Exprs)>1 时加括号，实测产出
	//      `WHERE u.del_flag='0' AND d.dept_id = ? OR u.user_id = ?`
	//      —— **缺外层括号**！AND 优先级高于 OR，实际语义变成
	//      (del_flag AND dept) OR user_id，会绕过 del_flag 过滤导致越权可见。
	//
	// 单条件时不能用 OrConditions：clause/where.go:48-52 对
	// `OrConditions{len(Exprs)==1}` 会按 " OR " 拼接，反而把 AND 变成 OR。
	return func(tx *gorm.DB) *gorm.DB {
		if len(condsCopy) == 1 {
			return tx.Where(condsCopy[0].sql, condsCopy[0].args...)
		}
		exprs := make([]clause.Expression, 0, len(condsCopy))
		for _, cc := range condsCopy {
			exprs = append(exprs, clause.Expr{SQL: cc.sql, Vars: cc.args})
		}
		// OrConditions.Build 在 len(Exprs)>1 时自动补外层括号 -> " AND (a OR b)"
		return tx.Where(clause.OrConditions{Exprs: exprs})
	}, ScopeInfo{Applied: true, Reason: "按数据范围过滤", Branches: branches, RolePermMissed: missed}
}

// noopScope admin / 全部数据权限：不追加任何条件。
func noopScope() func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB { return tx }
}

// denyScope 兜底：dept 字段 = 0，等价于 Java 的 " OR d.dept_id = 0"。
func denyScope(deptCol string) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB { return tx.Where(deptCol + " = 0") }
}

// rolePermOK 等价 L69/L81 的
// StringUtils.isEmpty(permission) || containsAny(role.getPermissions(), toStrArray(permission))。
func rolePermOK(r *entity.SysRole, permission string, perms []string) bool {
	if strings.TrimSpace(permission) == "" {
		return true
	}
	return r.HasPermission(perms...)
}

// splitPermission 等价 Convert.toStrArray(permission)：按 Constants.PERMISSION_DELIMITER 拆分。
func splitPermission(permission string) []string {
	out := make([]string, 0, 1)
	for _, p := range strings.Split(permission, config.PermissionDelimiter) {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// column 拼接 "别名.字段"；别名非法或为空时只返回字段名。
func column(alias, field string) string {
	if alias == "" || !aliasPattern.MatchString(alias) {
		return field
	}
	return alias + "." + field
}

func userIDOf(u *entity.LoginUser) int64 {
	if u == nil {
		return 0
	}
	return u.UserID
}

// deptIDOf 取用户部门 ID；为 nil 时按 0 处理。
//
// DIFF(review): Java 侧 user.getDeptId() 为 null 时会把 "null" 拼进 SQL
// （d.dept_id = null），Go 侧显式传 0，条件恒不成立但不会产生畸形 SQL。
func deptIDOf(u *entity.SysUser) int64 {
	if u == nil || u.DeptID == nil {
		return 0
	}
	return *u.DeptID
}
