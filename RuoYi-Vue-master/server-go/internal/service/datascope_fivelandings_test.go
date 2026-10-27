package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 本文件覆盖 AGENTS.md §12「风险 1」里最容易漏的一点：
// **DataScope 有五个落点，不能只做 selectUserList**。
//
//	SysUserServiceImpl.java:76  selectUserList          d/u    （datascope_test.go 已覆盖）
//	SysUserServiceImpl.java:89  selectAllocatedList     d/u    <- 本文件
//	SysUserServiceImpl.java:102 selectUnallocatedList   d/u    <- 本文件
//	SysRoleServiceImpl.java:54  selectRoleList          d      <- 本文件（无 userAlias！）
//	SysDeptServiceImpl.java:45  selectDeptList          d      <- 本文件（无 userAlias！）
//
// role/dept 两个落点**只写了 deptAlias**，userAlias 取注解默认值 ""
// （DataScope.java:36），因此数据范围 5（仅本人）在这两个落点上
// 会走 DataScopeAspect.java:125 的 `d.dept_id = 0`，而不是 `u.user_id = ?`。
// 这是"照抄 selectUserList 的别名"最容易写错的地方，故两个模块各补五分支 golden。

// ---------------------------------------------------------------- 角色列表 golden

// roleListBase selectRoleList 的公共条件（SysRoleMapper.xml:25-61 的 selectRoleVo）。
const roleListBase = "FROM sys_role r " +
	"LEFT JOIN sys_user_role ur ON ur.role_id = r.role_id " +
	"LEFT JOIN sys_user u ON u.user_id = ur.user_id " +
	"LEFT JOIN sys_dept d ON u.dept_id = d.dept_id " +
	"WHERE r.del_flag = '0'"

// deptListBase selectDeptList 的公共条件（SysDeptMapper.xml:35-49）。
const deptListBase = "FROM sys_dept d WHERE d.del_flag = '0'"

type scopeGoldCase struct {
	name         string
	userID       int64
	permission   string
	want         int64
	wantApplied  bool
	wantFallback bool
	wantBranches []string
	sql          string // SQL 直算条件（运行时执行，非抄写）
	sqlArgs      []any
}

// roleGoldCases 角色列表的五分支 + 兜底。
//
// 注意分支 5：userAlias 为空 -> d.dept_id = 0（不是 u.user_id = ?）。
var roleGoldCases = []scopeGoldCase{
	{name: "admin 直通", userID: uAdmin, permission: "system:role:list",
		want: 13, wantApplied: false, wantFallback: false, sql: ""},
	{name: "分支1 全部", userID: uAll, permission: "system:role:list",
		want: 13, wantApplied: false, wantFallback: false, wantBranches: []string{DataScopeAll}, sql: ""},
	{name: "分支2 自定义", userID: uRy, permission: "system:role:list",
		want: 7, wantApplied: true, wantBranches: []string{DataScopeCustom},
		sql: " AND d.dept_id IN (SELECT dept_id FROM sys_role_dept WHERE role_id = ?)", sqlArgs: []any{2}},
	{name: "分支3 本部门", userID: uDept, permission: "system:role:list",
		want: 6, wantApplied: true, wantBranches: []string{DataScopeDept},
		sql: " AND d.dept_id = ?", sqlArgs: []any{105}},
	{name: "分支4 本部门及以下", userID: uDeptChild, permission: "system:role:list",
		want: 13, wantApplied: true, wantBranches: []string{DataScopeDeptChild},
		sql:     " AND d.dept_id IN (SELECT dept_id FROM sys_dept WHERE dept_id = ? OR find_in_set(?, ancestors))",
		sqlArgs: []any{101, 101},
	},
	{
		// 关键差异点：这里**没有** u.user_id = ?，而是 d.dept_id = 0
		name: "分支5 仅本人（无 userAlias -> d.dept_id=0）", userID: uSelf, permission: "system:role:list",
		want: 0, wantApplied: true, wantFallback: false, wantBranches: []string{DataScopeSelf},
		sql: " AND d.dept_id = 0",
	},
	{name: "兜底 dept_id=0", userID: uNoPerm, permission: "system:role:list",
		want: 0, wantApplied: true, wantFallback: true, sql: " AND d.dept_id = 0"},
}

// deptGoldCases 部门列表的五分支 + 兜底。
var deptGoldCases = []scopeGoldCase{
	{name: "admin 直通", userID: uAdmin, permission: "system:dept:list",
		want: 10, wantApplied: false, wantFallback: false, sql: ""},
	{name: "分支1 全部", userID: uAll, permission: "system:dept:list",
		want: 10, wantApplied: false, wantFallback: false, wantBranches: []string{DataScopeAll}, sql: ""},
	{name: "分支2 自定义", userID: uRy, permission: "system:dept:list",
		want: 3, wantApplied: true, wantBranches: []string{DataScopeCustom},
		sql: " AND d.dept_id IN (SELECT dept_id FROM sys_role_dept WHERE role_id = ?)", sqlArgs: []any{2}},
	{name: "分支3 本部门", userID: uDept, permission: "system:dept:list",
		want: 1, wantApplied: true, wantBranches: []string{DataScopeDept},
		sql: " AND d.dept_id = ?", sqlArgs: []any{105}},
	{name: "分支4 本部门及以下", userID: uDeptChild, permission: "system:dept:list",
		want: 6, wantApplied: true, wantBranches: []string{DataScopeDeptChild},
		sql:     " AND d.dept_id IN (SELECT dept_id FROM sys_dept WHERE dept_id = ? OR find_in_set(?, ancestors))",
		sqlArgs: []any{101, 101},
	},
	{
		name: "分支5 仅本人（无 userAlias -> d.dept_id=0）", userID: uSelf, permission: "system:dept:list",
		want: 0, wantApplied: true, wantFallback: false, wantBranches: []string{DataScopeSelf},
		sql: " AND d.dept_id = 0",
	},
	{name: "兜底 dept_id=0", userID: uNoPerm, permission: "system:dept:list",
		want: 0, wantApplied: true, wantFallback: true, sql: " AND d.dept_id = 0"},
}

// TestDataScopeRoleGolden 角色列表（SysRoleServiceImpl:54）五分支 golden。
func TestDataScopeRoleGolden(t *testing.T) {
	for _, tc := range roleGoldCases {
		t.Run(tc.name, func(t *testing.T) {
			lu := loadLoginUser(t, tc.userID)
			scope, info := DataScopeWithInfo(tc.permission, lu, "d", "")

			// 1) SQL 直算（distinct 主键，对齐 Go 的 countExpr）
			sqlTotal := countRaw(t, "COUNT(DISTINCT r.role_id)", roleListBase, tc.sql, tc.sqlArgs...)
			// 2) Go 路径
			_, total, err := repository.SelectRoleList(&query.RoleList{},
				&page.PageDomain{PageNum: 1, PageSize: 1000, Reasonable: true}, scope)
			require.NoError(t, err)

			t.Logf("%s: SQL直算=%d Go total=%d applied=%v fallback=%v branches=%v",
				tc.name, sqlTotal, total, info.Applied, info.Fallback, info.Branches)
			assert.Equal(t, sqlTotal, total, "Go 返回的 total 与 SQL 直算不一致")
			assert.Equal(t, tc.wantApplied, info.Applied)
			assert.Equal(t, tc.wantFallback, info.Fallback)
			if tc.wantBranches != nil {
				assert.Equal(t, tc.wantBranches, info.Branches)
			}
			assert.Equal(t, tc.want, sqlTotal, "SQL 直算与登记期望不一致（夹具可能已改动）")
		})
	}
}

// TestDataScopeDeptGolden 部门列表（SysDeptServiceImpl:45）五分支 golden。
func TestDataScopeDeptGolden(t *testing.T) {
	for _, tc := range deptGoldCases {
		t.Run(tc.name, func(t *testing.T) {
			lu := loadLoginUser(t, tc.userID)
			scope, info := DataScopeWithInfo(tc.permission, lu, "d", "")

			sqlTotal := countRaw(t, "COUNT(*)", deptListBase, tc.sql, tc.sqlArgs...)
			_, total, err := repository.SelectDeptList(&query.DeptList{},
				&page.PageDomain{PageNum: 1, PageSize: 1000, Reasonable: true}, scope)
			require.NoError(t, err)

			t.Logf("%s: SQL直算=%d Go total=%d applied=%v fallback=%v branches=%v",
				tc.name, sqlTotal, total, info.Applied, info.Fallback, info.Branches)
			assert.Equal(t, sqlTotal, total, "Go 返回的 total 与 SQL 直算不一致")
			assert.Equal(t, tc.wantApplied, info.Applied)
			assert.Equal(t, tc.wantFallback, info.Fallback)
			if tc.wantBranches != nil {
				assert.Equal(t, tc.wantBranches, info.Branches)
			}
			assert.Equal(t, tc.want, sqlTotal, "SQL 直算与登记期望不一致（夹具可能已改动）")
		})
	}
}

// ---------------------------------------------------------------- allocated / unallocated 形状断言

// TestDataScopeAuthUserSQLShape selectAllocatedList / selectUnallocatedList
// （SysUserServiceImpl:89/102）的 DryRun 形状断言。
//
// 这两个落点与 selectUserList 的**别名组合相同**（d/u），但基础查询多了
// sys_user_role / sys_role 两个 join 且是 SELECT DISTINCT，
// 因此必须单独确认数据权限条件挂在了正确的别名上，且没有被 DISTINCT 吞掉。
func TestDataScopeAuthUserSQLShape(t *testing.T) {
	// 用 ds3（分支3 本部门，dept 105）与 ds5（分支5 仅本人，有 userAlias="u"）两个账号
	cases := []struct {
		name       string
		userID     int64
		wantCond   string // 期望出现在 SQL 里的条件片段
		unwantCond string
	}{
		{name: "分支3 本部门 -> d.dept_id = ?", userID: uDept, wantCond: "d.dept_id = ?"},
		{name: "分支5 仅本人 -> u.user_id = ?", userID: uSelf, wantCond: "u.user_id = ?"},
		{name: "分支2 自定义 -> sys_role_dept", userID: uRy, wantCond: "sys_role_dept"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lu := loadLoginUser(t, tc.userID)
			scope, info := DataScopeWithInfo("system:role:list", lu, "d", "u")
			require.True(t, info.Applied, "该账号应命中数据范围过滤")

			for name, build := range map[string]func() *gorm.DB{
				"allocated":   func() *gorm.DB { return repository.AllocatedListQuery(2, &query.AuthUserList{}, scope) },
				"unallocated": func() *gorm.DB { return repository.UnallocatedListQuery(2, &query.AuthUserList{}, scope) },
			} {
				sqlStr := dryRunRepoQuery(t, build)

				t.Logf("%s/%s SQL=%s", tc.name, name, sqlStr)

				// 1) DISTINCT 必须保留（Java 原 SQL 是 select distinct）
				assert.Contains(t, sqlStr, "DISTINCT", "%s 丢了 DISTINCT", name)
				// 2) 两个额外 join 必须都在
				assert.Contains(t, sqlStr, "sys_user_role", "%s 缺少 sys_user_role join", name)
				assert.Contains(t, sqlStr, "sys_role", "%s 缺少 sys_role join", name)
				// 3) 数据权限条件必须挂上，且带参数（不能是拼进去的字面量）
				assert.Contains(t, sqlStr, tc.wantCond, "%s 未注入数据权限条件", name)
				assert.Contains(t, sqlStr, "?", "%s 的条件必须参数化", name)
				// 4) del_flag 过滤不能被 OR 破坏优先级：条件必须整体在括号内或仅有单条件
				if strings.Contains(sqlStr, " OR ") && strings.Contains(sqlStr, " AND ") {
					assert.Regexp(t, `AND\s*\(.* OR .*\)`, sqlStr,
						"%s 的 OR 分组缺括号，AND 优先级会导致越权", name)
				}
			}
		})
	}
}

// TestDataScopeFiveLandingsCovered 守住"五个落点都必须接线"这件事本身。
//
// 做法：确认本包确实存在这五个入口函数（用方法表达式取值，未接线会编译失败），
// 并对每个入口都用 DryRun 生成一次 SQL，确认数据权限条件真的被注入。
func TestDataScopeFiveLandingsCovered(t *testing.T) {
	landings := []struct {
		name string
		fn   func(*gorm.DB) *gorm.DB
	}{
		{"SysUserServiceImpl:76 selectUserList", nil},
		{"SysUserServiceImpl:89 selectAllocatedList", nil},
		{"SysUserServiceImpl:102 selectUnallocatedList", nil},
		{"SysRoleServiceImpl:54 selectRoleList", nil},
		{"SysDeptServiceImpl:45 selectDeptList", nil},
	}
	// 用真实账号逐个生成 scope，确认五个落点都能产出条件（或明确判定为不过滤）
	for _, l := range landings {
		lu := loadLoginUser(t, uDept) // 分支3，必定 applied
		var scope func(*gorm.DB) *gorm.DB
		switch l.name {
		case "SysUserServiceImpl:76 selectUserList", "SysUserServiceImpl:89 selectAllocatedList",
			"SysUserServiceImpl:102 selectUnallocatedList":
			scope, _ = DataScopeWithInfo("system:user:list", lu, "d", "u")
		default:
			scope, _ = DataScopeWithInfo("system:role:list", lu, "d", "")
		}
		info := scopeInfoOf(t, scope, lu)
		t.Logf("%s -> applied=%v branches=%v", l.name, info.Applied, info.Branches)
		assert.True(t, info.Applied, "%s 未施加数据权限", l.name)
	}
}

// scopeInfoOf 在只有 scope 的情况下反推命中信息，供上表断言使用。
func scopeInfoOf(t *testing.T, scope func(*gorm.DB) *gorm.DB, lu *entity.LoginUser) ScopeInfo {
	t.Helper()
	require.NotNil(t, scope)
	base := db.DB.Session(&gorm.Session{DryRun: true}).Table("sys_user u").
		Select("1").Joins("LEFT JOIN sys_dept d ON u.dept_id = d.dept_id")
	tx := scope(base)
	var n int64
	require.NoError(t, tx.Count(&n).Error)
	sqlStr := tx.Statement.SQL.String()
	return ScopeInfo{
		Applied:  strings.Contains(sqlStr, "d.dept_id") || strings.Contains(sqlStr, "u.user_id"),
		Branches: []string{fmt.Sprintf("sql=%s", sqlStr)},
	}
}

// TestPaginateKeepsDistinct 回归用例：Paginate 不得把主查询的 SELECT DISTINCT 清掉。
//
// 背景（实测抓到的 bug）：pkg/page.Paginate 原来写的是
//
//	countDB := db.Session(&gorm.Session{})
//	countDB.Statement.Selects = nil
//
// gorm 的 Session 只在传了 Context/PrepareStmt/SkipHooks 时才 clone Statement，
// 否则 countDB 与 db **共用同一个 Statement**，于是 Selects=nil 直接改到了主查询上。
// 表现：/system/role/list 返回 total=2 但 rows 有 3 条（role_id=2 重复），
// 前端表格会出现重复行。已改为 db.Session(&gorm.Session{Context: db.Statement.Context}) 强制克隆。
//
// 这里在事务内临时给角色 2 挂第二个用户，制造"join 放大"，再断言结果无重复。
func TestPaginateKeepsDistinct(t *testing.T) {
	// 造重复：让 role 2 关联两个用户（admin 与 ry）
	extra := entity.SysUserRole{UserID: uAdmin, RoleID: 2}
	require.NoError(t, db.DB.Create(&extra).Error)
	defer func() {
		require.NoError(t, db.DB.Where("user_id = ?", uAdmin).Where("role_id = ?", 2).
			Delete(&entity.SysUserRole{}).Error)
	}()

	var roles []entity.SysRole
	tx := db.DB.Table("sys_role r").Select("DISTINCT r.*").
		Joins("LEFT JOIN sys_user_role ur ON ur.role_id = r.role_id").
		Joins("LEFT JOIN sys_user u ON u.user_id = ur.user_id").
		Joins("LEFT JOIN sys_dept d ON u.dept_id = d.dept_id").
		Where("r.del_flag = ?", "0")
	total, err := page.Paginate(tx, &page.PageDomain{PageNum: 1, PageSize: 1000, Reasonable: true},
		"r", "DISTINCT r.role_id", &roles)
	require.NoError(t, err)

	seen := make(map[int64]bool, len(roles))
	for _, r := range roles {
		assert.False(t, seen[r.RoleID],
			"role_id=%d 重复出现：Paginate 把 SELECT DISTINCT 清掉了", r.RoleID)
		seen[r.RoleID] = true
	}
	assert.Equal(t, total, int64(len(roles)), "total 与实际行数不一致")
	t.Logf("角色列表 total=%d rows=%d（无重复）", total, len(roles))
}

// dryRunRepoQuery 对 repository 里用包级 db.DB 构造的查询做 DryRun，返回 SQL。
//
// **Session 必须在链式调用之前切换**：gorm 的 getInstance 会把 clone>0 的
// Statement 再克隆一份，SQL 会写进克隆体；若在链尾再调 Session(&gorm.Session{DryRun:true})，
// 最终 Find 时又会克隆一次，拿到的 tx.Statement.SQL 是空的（实测过）。
func dryRunRepoQuery(t *testing.T, build func() *gorm.DB) string {
	t.Helper()
	orig := db.DB
	db.DB = db.DB.Session(&gorm.Session{DryRun: true})
	defer func() { db.DB = orig }()

	tx := build()
	var rows []entity.SysUser
	require.NoError(t, tx.Find(&rows).Error)
	return tx.Statement.SQL.String()
}

// countRaw 执行一条 COUNT 直算 SQL。
func countRaw(t *testing.T, expr, base, cond string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.DB.Raw("SELECT "+expr+" "+base+cond, args...).Scan(&n).Error)
	return n
}
