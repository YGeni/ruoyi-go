package service

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 本文件是 AGENTS.md §12「风险 1」的规避手段之一：
// 五个数据范围分支 + 空条件兜底，逐个与 SQL 直算结果做 golden 比对。
//
// **零副作用设计（Q2）**：主库不写入任何测试数据。
//
//	TestMain 开启一个事务，在事务内插入全部夹具（账号/角色/关联），
//	并把 db.DB 临时指向该事务——repository 的所有查询因此都发生在事务内，
//	能看到未提交的夹具；m.Run() 结束后 Rollback，主库保持原状。
//
// 形状类断言（M1/M2 的 SQL 结构）一律走 DryRun，不产生任何查询。
//
// 运行前置：Docker 里的 ry-vue 库（scripts/setup.sh）。sys_dept（100-109）
// 与 admin/ry 两个账号 + role 1/2 依赖库中已提交的种子数据，夹具只增量补充。

var (
	testTx  *gorm.DB // 夹具所在事务
	origDB  *gorm.DB // 事务外的原始连接，用于校验"零副作用"
	testPwd string
)

func TestMain(m *testing.M) {
	if err := config.Load("../../config/config.yaml"); err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}
	if db.DB == nil {
		if err := db.Init(config.C.Datasource.Master); err != nil {
			fmt.Fprintf(os.Stderr, "连接 MySQL 失败，golden 测试需要真实库: %v\n", err)
			os.Exit(1)
		}
	}
	origDB = db.DB
	if err := origDB.Raw("SELECT password FROM sys_user WHERE user_id = 1").Scan(&testPwd).Error; err != nil {
		fmt.Fprintf(os.Stderr, "读取 admin 密码散列失败: %v\n", err)
		os.Exit(1)
	}

	tx := origDB.Begin()
	if tx.Error != nil {
		fmt.Fprintf(os.Stderr, "开启事务失败: %v\n", tx.Error)
		os.Exit(1)
	}
	if err := insertFixtures(tx); err != nil {
		_ = tx.Rollback()
		fmt.Fprintf(os.Stderr, "插入夹具失败: %v\n", err)
		os.Exit(1)
	}
	testTx = tx
	db.DB = tx // repository 全部走同一事务

	code := m.Run()

	db.DB = origDB
	_ = tx.Rollback()
	os.Exit(code)
}

func ptr[T any](v T) *T { return &v }

// ---------------------------------------------------------------- 夹具

// fixture 常量：全部使用显式主键，保证 SQL 直算里的字面量稳定。
//
//	用户  100-107 + 200/202/203（admin=1、ry=2 来自种子数据）
//	角色  100-104 + 110/111/120/121/130/131（role 1=admin、2=common 来自种子数据）
//
// 部门分布（决定各分支的期望行数）：
//
//	100:1  101:1  103:3  104:1  105:5  106:2   —— 合计 13
const (
	// 账号
	uAdmin, uRy = 1, 2
	uAll        = 100 // ds_all     dept 103  role ds_all(1)
	uDept       = 101 // ds3        dept 105  role ds3(3)
	uDeptChild  = 102 // ds4        dept 101  role ds4(4)
	uSelf       = 103 // ds5        dept 105  role ds5(5)
	uNoPerm     = 104 // ds_noperm  dept 105  role ds_noperm(2, 无 role_menu)
	uF100       = 105 // 填充 dept 100
	uF104       = 106 // 填充 dept 104
	uF106       = 107 // 填充 dept 106
	uMix        = 200 // M1 dept 105 roles: mixDept(3) + mixSelf(5)
	uTwoCustom  = 202 // M2 dept 103 roles: customA(2) + customB(2)
	uCustomAll  = 203 // M3 dept 106 roles: customC(2) 然后 allAfter(1)

	// 角色
	rAll       = 100 // data_scope 1
	rDept      = 101 // data_scope 3
	rDeptChild = 102 // data_scope 4
	rSelf      = 103 // data_scope 5
	rNoPerm    = 104 // data_scope 2，且**不挂任何 sys_role_menu**
	rMixDept   = 110 // data_scope 3
	rMixSelf   = 111 // data_scope 5
	rCustomA   = 120 // data_scope 2 -> sys_role_dept 105
	rCustomB   = 121 // data_scope 2 -> sys_role_dept 103
	rCustomC   = 130 // data_scope 2 -> sys_role_dept 105
	rAllAfter  = 131 // data_scope 1（M3：排在 customC 之后）

	// 每个测试角色都要挂上的菜单，否则 GetMenuPermission 查不到权限、
	// 角色会被 DataScope 的 permission 判断跳过（DataScopeAspect.java:81）。
	//
	// 三个都要挂：DataScope 的 permission 取当前接口的权限串，
	// 因此角色/部门列表的 golden 用例要求角色同时持有 system:role:list 与 system:dept:list。
	menuUserList = 100 // sys_menu.perms = system:user:list
	menuRoleList = 101 // sys_menu.perms = system:role:list
	menuDeptList = 103 // sys_menu.perms = system:dept:list

	// fixtureTag 夹具标记，写入 create_by；TestFixturesNotCommitted 用它断言零副作用。
	fixtureTag = "1b-fixture"
)

func insertFixtures(tx *gorm.DB) error {
	roles := []entity.SysRole{
		{RoleID: rAll, RoleName: "DS-全部", RoleKey: "ds_all", RoleSort: 90, DataScope: DataScopeAll, Status: config.Normal, DelFlag: model.DelFlagExists},
		{RoleID: rDept, RoleName: "DS-本部门", RoleKey: "ds3", RoleSort: 91, DataScope: DataScopeDept, Status: config.Normal, DelFlag: model.DelFlagExists},
		{RoleID: rDeptChild, RoleName: "DS-本部门及以下", RoleKey: "ds4", RoleSort: 92, DataScope: DataScopeDeptChild, Status: config.Normal, DelFlag: model.DelFlagExists},
		{RoleID: rSelf, RoleName: "DS-仅本人", RoleKey: "ds5", RoleSort: 93, DataScope: DataScopeSelf, Status: config.Normal, DelFlag: model.DelFlagExists},
		{RoleID: rNoPerm, RoleName: "DS-无权限串", RoleKey: "ds_noperm", RoleSort: 94, DataScope: DataScopeCustom, Status: config.Normal, DelFlag: model.DelFlagExists},
		{RoleID: rMixDept, RoleName: "M1-本部门", RoleKey: "mix_dept", RoleSort: 95, DataScope: DataScopeDept, Status: config.Normal, DelFlag: model.DelFlagExists},
		{RoleID: rMixSelf, RoleName: "M1-仅本人", RoleKey: "mix_self", RoleSort: 96, DataScope: DataScopeSelf, Status: config.Normal, DelFlag: model.DelFlagExists},
		{RoleID: rCustomA, RoleName: "M2-自定义A", RoleKey: "custom_a", RoleSort: 97, DataScope: DataScopeCustom, Status: config.Normal, DelFlag: model.DelFlagExists},
		{RoleID: rCustomB, RoleName: "M2-自定义B", RoleKey: "custom_b", RoleSort: 98, DataScope: DataScopeCustom, Status: config.Normal, DelFlag: model.DelFlagExists},
		{RoleID: rCustomC, RoleName: "M3-自定义在前", RoleKey: "custom_c", RoleSort: 99, DataScope: DataScopeCustom, Status: config.Normal, DelFlag: model.DelFlagExists},
		{RoleID: rAllAfter, RoleName: "M3-全部在后", RoleKey: "all_after", RoleSort: 100, DataScope: DataScopeAll, Status: config.Normal, DelFlag: model.DelFlagExists},
	}
	// BaseModel 是嵌入字段，无法在复合字面量里直接赋值，故循环设置；
	// create_by = '1b-fixture' 同时作为 TestFixturesNotCommitted 的识别标记。
	for i := range roles {
		roles[i].BaseModel.CreateBy = fixtureTag
		roles[i].BaseModel.CreateTime = ptr(time.Now())
	}
	if err := tx.Create(&roles).Error; err != nil {
		return fmt.Errorf("插入角色: %w", err)
	}

	// rNoPerm 故意不挂菜单 -> 权限集为空 -> 带 permission 时必然落到兜底分支
	menus := make([]entity.SysRoleMenu, 0, (len(roles)-1)*3)
	for _, r := range roles {
		if r.RoleID == rNoPerm {
			continue
		}
		for _, m := range []int64{menuUserList, menuRoleList, menuDeptList} {
			menus = append(menus, entity.SysRoleMenu{RoleID: r.RoleID, MenuID: m})
		}
	}
	if err := tx.Create(&menus).Error; err != nil {
		return fmt.Errorf("插入角色菜单: %w", err)
	}

	users := []entity.SysUser{
		{UserID: uAll, UserName: "ds_all", NickName: "DS-全部", UserType: "00", DeptID: ptr(int64(103)), Password: testPwd, Status: config.Normal, DelFlag: model.DelFlagExists},
		{UserID: uDept, UserName: "ds3", NickName: "DS-本部门", UserType: "00", DeptID: ptr(int64(105)), Password: testPwd, Status: config.Normal, DelFlag: model.DelFlagExists},
		{UserID: uDeptChild, UserName: "ds4", NickName: "DS-本部门及以下", UserType: "00", DeptID: ptr(int64(101)), Password: testPwd, Status: config.Normal, DelFlag: model.DelFlagExists},
		{UserID: uSelf, UserName: "ds5", NickName: "DS-仅本人", UserType: "00", DeptID: ptr(int64(105)), Password: testPwd, Status: config.Normal, DelFlag: model.DelFlagExists},
		{UserID: uNoPerm, UserName: "ds_noperm", NickName: "DS-无权限串", UserType: "00", DeptID: ptr(int64(105)), Password: testPwd, Status: config.Normal, DelFlag: model.DelFlagExists},
		{UserID: uF100, UserName: "ds_u100", NickName: "填充-100", UserType: "00", DeptID: ptr(int64(100)), Password: testPwd, Status: config.Normal, DelFlag: model.DelFlagExists},
		{UserID: uF104, UserName: "ds_u104", NickName: "填充-104", UserType: "00", DeptID: ptr(int64(104)), Password: testPwd, Status: config.Normal, DelFlag: model.DelFlagExists},
		{UserID: uF106, UserName: "ds_u106", NickName: "填充-106", UserType: "00", DeptID: ptr(int64(106)), Password: testPwd, Status: config.Normal, DelFlag: model.DelFlagExists},
		{UserID: uMix, UserName: "ds_mix", NickName: "M1-混合", UserType: "00", DeptID: ptr(int64(105)), Password: testPwd, Status: config.Normal, DelFlag: model.DelFlagExists},
		{UserID: uTwoCustom, UserName: "ds_two_custom", NickName: "M2-双自定义", UserType: "00", DeptID: ptr(int64(103)), Password: testPwd, Status: config.Normal, DelFlag: model.DelFlagExists},
		{UserID: uCustomAll, UserName: "ds_custom_then_all", NickName: "M3-自定义后全部", UserType: "00", DeptID: ptr(int64(106)), Password: testPwd, Status: config.Normal, DelFlag: model.DelFlagExists},
	}
	for i := range users {
		users[i].BaseModel.CreateBy = fixtureTag
		users[i].BaseModel.CreateTime = ptr(time.Now())
	}
	if err := tx.Create(&users).Error; err != nil {
		return fmt.Errorf("插入用户: %w", err)
	}

	userRoles := []entity.SysUserRole{
		{UserID: uAll, RoleID: rAll},
		{UserID: uDept, RoleID: rDept},
		{UserID: uDeptChild, RoleID: rDeptChild},
		{UserID: uSelf, RoleID: rSelf},
		{UserID: uNoPerm, RoleID: rNoPerm},
		{UserID: uMix, RoleID: rMixDept}, {UserID: uMix, RoleID: rMixSelf},
		{UserID: uTwoCustom, RoleID: rCustomA}, {UserID: uTwoCustom, RoleID: rCustomB},
		// M3：顺序即断言前提——customC(2) 在前、allAfter(1) 在后
		{UserID: uCustomAll, RoleID: rCustomC}, {UserID: uCustomAll, RoleID: rAllAfter},
	}
	if err := tx.Create(&userRoles).Error; err != nil {
		return fmt.Errorf("插入用户角色: %w", err)
	}

	roleDepts := []entity.SysRoleDept{
		{RoleID: rCustomA, DeptID: 105},
		{RoleID: rCustomB, DeptID: 103},
		{RoleID: rCustomC, DeptID: 105},
	}
	return tx.Create(&roleDepts).Error
}

// ---------------------------------------------------------------- 辅助

// userListBase 用户列表的公共条件，对应 SysUserMapper.xml:59-83 的 where u.del_flag='0'。
const userListBase = "FROM sys_user u LEFT JOIN sys_dept d ON u.dept_id = d.dept_id WHERE u.del_flag = '0'"

// loadLoginUser 从事务内装载账号并回填角色权限，
// 与登录链路（createToken -> GetMenuPermission）完全同款，
// 确保 SysRole.Permissions 被填充（否则会静默落到兜底分支）。
func loadLoginUser(t *testing.T, userID int64) *entity.LoginUser {
	t.Helper()
	u, err := repository.SelectUserByID(userID)
	require.NoError(t, err)
	require.NotNil(t, u, "userID=%d 不存在（夹具未生效？）", userID)
	perms, err := GetMenuPermission(u)
	require.NoError(t, err)
	return &entity.LoginUser{UserID: u.UserID, DeptID: u.DeptID, Permissions: perms, User: u}
}

// countUsers 走真实 GORM 查询拿 total（事务内）。
func countUsers(t *testing.T, lu *entity.LoginUser, permission string) (int64, ScopeInfo) {
	t.Helper()
	scope, info := DataScopeWithInfo(permission, lu, "d", "u")
	_, total, err := repository.SelectUserList(&query.UserList{},
		&page.PageDomain{PageNum: 1, PageSize: 1000, Reasonable: true}, scope)
	require.NoError(t, err)
	return total, info
}

// sqlCount SQL 直算：独立于 Go 实现，用字面量拼出与 Java 同形的条件。
func sqlCount(t *testing.T, where string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.DB.Raw("SELECT COUNT(*) "+userListBase+where, args...).Scan(&n).Error)
	return n
}

// dryRunSQL 生成 selectUserList 的完整 SQL 但不执行（零副作用）。
func dryRunSQL(t *testing.T, lu *entity.LoginUser, permission, deptAlias, userAlias string) (string, []any) {
	t.Helper()
	scope, _ := DataScopeWithInfo(permission, lu, deptAlias, userAlias)
	tx := db.DB.Session(&gorm.Session{DryRun: true}).
		Table("sys_user u").Select("u.*").
		Joins("LEFT JOIN sys_dept d ON u.dept_id = d.dept_id").
		// 必须与真实查询一致地带 u.del_flag，否则无法暴露 OR 优先级问题
		Where("u.del_flag = ?", model.DelFlagExists)
	if scope != nil {
		tx = tx.Scopes(scope)
	}
	var rows []entity.SysUser
	require.NoError(t, tx.Find(&rows).Error)
	return tx.Statement.SQL.String(), tx.Statement.Vars
}

// ---------------------------------------------------------------- golden

type goldCase struct {
	name         string
	userID       int64
	permission   string
	want         int64 // SQL 直算 + 人工核对的期望行数
	wantApplied  bool
	wantFallback bool
	wantBranches []string
	sql          string // SQL 直算条件（运行时执行，非抄写）
	sqlArgs      []any
}

var goldCases = []goldCase{
	{
		name: "admin 直通（不过滤）", userID: uAdmin, permission: "system:user:list",
		want: 13, wantApplied: false, wantFallback: false,
		sql: "",
	},
	{
		name: "分支1 全部数据权限（非 admin 账号）", userID: uAll, permission: "system:user:list",
		want: 13, wantApplied: false, wantFallback: false, wantBranches: []string{DataScopeAll},
		sql: "",
	},
	{
		name: "分支2 自定义（role_id=2 -> sys_role_dept 100,101,105）", userID: uRy, permission: "system:user:list",
		want: 7, wantApplied: true, wantFallback: false, wantBranches: []string{DataScopeCustom},
		sql: " AND d.dept_id IN (SELECT dept_id FROM sys_role_dept WHERE role_id = ?)", sqlArgs: []any{2},
	},
	{
		name: "分支3 本部门（dept_id=105）", userID: uDept, permission: "system:user:list",
		want: 5, wantApplied: true, wantFallback: false, wantBranches: []string{DataScopeDept},
		sql: " AND d.dept_id = ?", sqlArgs: []any{105},
	},
	{
		name: "分支4 本部门及以下（dept_id=101 + ancestors）", userID: uDeptChild, permission: "system:user:list",
		want: 12, wantApplied: true, wantFallback: false, wantBranches: []string{DataScopeDeptChild},
		sql: " AND d.dept_id IN (SELECT dept_id FROM sys_dept WHERE dept_id = ? OR find_in_set(?, ancestors))", sqlArgs: []any{101, 101},
	},
	{
		name: "分支5 仅本人（userAlias=u -> u.user_id=103）", userID: uSelf, permission: "system:user:list",
		want: 1, wantApplied: true, wantFallback: false, wantBranches: []string{DataScopeSelf},
		sql: " AND u.user_id = ?", sqlArgs: []any{103},
	},
	{
		name: "兜底 data_scope=2 但角色未持有 permission -> dept_id=0", userID: uNoPerm, permission: "system:user:list",
		want: 0, wantApplied: true, wantFallback: true,
		sql: " AND d.dept_id = 0",
	},
}

func TestDataScopeGolden(t *testing.T) {
	for _, tc := range goldCases {
		t.Run(tc.name, func(t *testing.T) {
			lu := loadLoginUser(t, tc.userID)

			// 1) SQL 直算
			sqlTotal := sqlCount(t, tc.sql, tc.sqlArgs...)
			// 2) Go 路径
			total, info := countUsers(t, lu, tc.permission)

			t.Logf("account=%s(userID=%d) SQL直算=%d Go total=%d applied=%v fallback=%v branches=%v",
				lu.User.UserName, tc.userID, sqlTotal, total, info.Applied, info.Fallback, info.Branches)

			assert.Equal(t, tc.want, sqlTotal, "SQL 直算与登记期望不一致，夹具可能已改动")
			assert.Equal(t, sqlTotal, total, "Go 返回的 total 与 SQL 直算不一致")
			assert.Equal(t, tc.wantApplied, info.Applied)
			assert.Equal(t, tc.wantFallback, info.Fallback)
			if tc.wantBranches != nil {
				assert.Equal(t, tc.wantBranches, info.Branches)
			}
			if tc.wantFallback {
				assert.NotEmpty(t, info.Reason, "兜底必须给出可读原因")
			}
		})
	}
}

// TestDataScopeGoldenReport 输出可直接贴进报告的比对表。
func TestDataScopeGoldenReport(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("账号\t分支\ttotal\tSQL直算\t一致\n")
	for _, tc := range goldCases {
		lu := loadLoginUser(t, tc.userID)
		total, info := countUsers(t, lu, tc.permission)
		sqlTotal := sqlCount(t, tc.sql, tc.sqlArgs...)
		branch := "-"
		if len(info.Branches) > 0 {
			branch = strings.Join(info.Branches, ",")
		}
		if info.Fallback {
			branch += " (兜底)"
		}
		sb.WriteString(fmt.Sprintf("%s(%d)\t%s\t%d\t%d\t%v\n",
			lu.User.UserName, tc.userID, branch, total, sqlTotal, total == sqlTotal))
	}
	fmt.Print("\n" + sb.String())
}

// ---------------------------------------------------------------- M1/M2/M3

// TestDataScopeM1MultiRoleMixed 多角色混合：roleA=3 + roleB=5。
//
// 依据 DataScopeAspect.java:108-128 —— 两个分支各追加一段 " OR ..."，
// 最终由 L143 拼成 " AND (d.dept_id = ? OR u.user_id = ?)"。
// 断言：SQL 形状为 OR 并存（不是只取第一个分支），且行数按**并集**计算。
func TestDataScopeM1MultiRoleMixed(t *testing.T) {
	lu := &entity.LoginUser{
		UserID: uMix,
		DeptID: ptr(int64(105)),
		User: &entity.SysUser{
			UserID: uMix,
			DeptID: ptr(int64(105)),
			// 顺序：本部门(3) 在前、仅本人(5) 在后
			Roles: []entity.SysRole{
				{RoleID: rMixDept, RoleKey: "mix_dept", DataScope: DataScopeDept, Status: config.Normal, Permissions: []string{"system:user:list"}},
				{RoleID: rMixSelf, RoleKey: "mix_self", DataScope: DataScopeSelf, Status: config.Normal, Permissions: []string{"system:user:list"}},
			},
		},
	}

	// --- SQL 形状（DryRun，零副作用）---
	sqlStr, vars := dryRunSQL(t, lu, "system:user:list", "d", "u")
	t.Logf("M1 SQL=%s VARS=%v", sqlStr, vars)
	assert.Contains(t, sqlStr, "d.dept_id = ?", "必须保留分支 3 的条件")
	assert.Contains(t, sqlStr, "u.user_id = ?", "必须保留分支 5 的条件")
	assert.Contains(t, sqlStr, " OR ", "两个分支必须以 OR 并存，不能被后续分支覆盖")
	// vars[0] 是 dryRunSQL 自带的 u.del_flag，其后才是数据权限条件参数
	require.Len(t, vars, 3, "del_flag + dept_id + user_id")
	assert.EqualValues(t, 105, vars[1])
	assert.EqualValues(t, uMix, vars[2])
	// 分组括号：AND ( ... OR ... )
	assert.Regexp(t, `\(\s*d\.dept_id = \?\s+OR\s+u\.user_id = \?\s*\)`, sqlStr,
		"必须包在一对括号里，避免与外层 AND 优先级混淆")

	// --- 行数按并集断言 ---
	total, info := countUsers(t, lu, "system:user:list")
	assert.Equal(t, []string{DataScopeDept, DataScopeSelf}, info.Branches)
	// 分支3 命中 dept 105 -> 5 人；分支5 命中 user_id=200 -> 1 人；200 本身在 105，并集 = 5
	wantDept := sqlCount(t, " AND d.dept_id = ?", 105)
	wantSelf := sqlCount(t, " AND u.user_id = ?", uMix)
	wantUnion := sqlCount(t, " AND (d.dept_id = ? OR u.user_id = ?)", 105, uMix)
	t.Logf("M1 分支3=%d 分支5=%d 并集=%d Go total=%d", wantDept, wantSelf, wantUnion, total)
	assert.Equal(t, int64(5), wantDept)
	assert.Equal(t, int64(1), wantSelf)
	assert.Equal(t, int64(5), wantUnion, "200 号用户本就属于 dept 105，并集应为 5")
	assert.Equal(t, wantUnion, total, "多角色必须取并集，不能只取单一分支")
}

// TestDataScopeM2TwoCustomRoles 两个自定义角色必须合并成**单条** role_id IN (?,?)。
//
// 依据 DataScopeAspect.java:96-106：scopeCustomIds.size() > 1 时走 in 查询
// （注释原话「多个自定数据权限使用in查询，避免多次拼接」），且 L77 的去重保证
// 第二个 data_scope=2 角色不会再追加一段 OR。
func TestDataScopeM2TwoCustomRoles(t *testing.T) {
	lu := &entity.LoginUser{
		UserID: uTwoCustom,
		DeptID: ptr(int64(103)),
		User: &entity.SysUser{
			UserID: uTwoCustom,
			DeptID: ptr(int64(103)),
			Roles: []entity.SysRole{
				{RoleID: rCustomA, RoleKey: "custom_a", DataScope: DataScopeCustom, Status: config.Normal, Permissions: []string{"system:user:list"}},
				{RoleID: rCustomB, RoleKey: "custom_b", DataScope: DataScopeCustom, Status: config.Normal, Permissions: []string{"system:user:list"}},
			},
		},
	}

	sqlStr, vars := dryRunSQL(t, lu, "system:user:list", "d", "u")
	t.Logf("M2 SQL=%s VARS=%v", sqlStr, vars)

	// 形状：只有一条 IN 子查询，且是 role_id IN (?,?)（两个占位符）
	assert.Equal(t, 1, strings.Count(sqlStr, "sys_role_dept"), "两个自定义角色只能生成一条子查询")
	assert.Contains(t, sqlStr, "role_id IN (?,?)", "必须合并为 IN (?,?) 而不是两条 OR")
	assert.NotContains(t, sqlStr, " OR ", "单分支时不该出现 OR")
	assert.Contains(t, sqlStr, "sys_role_dept WHERE role_id IN (?,?)")
	// GORM 会把 IN 的切片参数摊平进 Statement.Vars，故这里是两个值
	// vars[0] 是 u.del_flag；其后两个是 IN (?,?) 的两个 role_id
	require.Len(t, vars, 3, "del_flag + 两个 role_id")
	assert.EqualValues(t, rCustomA, vars[1])
	assert.EqualValues(t, rCustomB, vars[2])

	// 行数：depts = {105(custom_a), 103(custom_b)} -> 5 + 3 = 8
	total, info := countUsers(t, lu, "system:user:list")
	want := sqlCount(t, " AND d.dept_id IN (SELECT dept_id FROM sys_role_dept WHERE role_id IN (?,?))", rCustomA, rCustomB)
	t.Logf("M2 期望=%d Go total=%d branches=%v", want, total, info.Branches)
	assert.Equal(t, int64(8), want)
	assert.Equal(t, want, total)
	assert.Equal(t, []string{DataScopeCustom}, info.Branches, "去重后只应命中一次分支 2")
}

// TestDataScopeM3CustomThenAll 自定义在前、全部在后：Java 清空已拼条件并 break。
//
// 依据 DataScopeAspect.java:90-95 + 131-145：
//   - L92  sqlString = new StringBuilder()  // 清空已拼的自定义条件
//   - L93  conditions.add(dataScope)        // conditions 非空，不会走 L132 兜底
//   - L94  break
//   - L137 StringUtils.isNotBlank(sqlString) == false -> **不注入任何条件**
//
// 因此 Go 侧必须是 applied=false（无过滤、返回全量），
// 既不能残留自定义条件，也不能退化成 dept_id=0 兜底。
func TestDataScopeM3CustomThenAll(t *testing.T) {
	lu := &entity.LoginUser{
		UserID: uCustomAll,
		DeptID: ptr(int64(106)),
		User: &entity.SysUser{
			UserID: uCustomAll,
			DeptID: ptr(int64(106)),
			// 顺序即断言前提：自定义(2) 在前、全部(1) 在后
			Roles: []entity.SysRole{
				{RoleID: rCustomC, RoleKey: "custom_c", DataScope: DataScopeCustom, Status: config.Normal, Permissions: []string{"system:user:list"}},
				{RoleID: rAllAfter, RoleKey: "all_after", DataScope: DataScopeAll, Status: config.Normal, Permissions: []string{"system:user:list"}},
			},
		},
	}

	sqlStr, vars := dryRunSQL(t, lu, "system:user:list", "d", "u")
	t.Logf("M3 SQL=%s VARS=%v", sqlStr, vars)

	// 不能残留自定义条件，也不能出现兜底条件
	assert.NotContains(t, sqlStr, "sys_role_dept", "ALL 分支必须清空已拼的自定义条件")
	assert.NotContains(t, sqlStr, "dept_id = 0", "不能退化成兜底")
	assert.NotContains(t, sqlStr, "WHERE d.dept_id", "不应残留任何 dept 过滤")
	assert.Len(t, vars, 1, "ALL 分支不应追加任何参数，只剩 u.del_flag")

	total, info := countUsers(t, lu, "system:user:list")
	t.Logf("M3 total=%d applied=%v fallback=%v branches=%v", total, info.Applied, info.Fallback, info.Branches)
	assert.False(t, info.Applied, "命中 ALL 后应无过滤")
	assert.False(t, info.Fallback, "conditions 非空（Java L93 已 add），不是兜底")
	assert.Equal(t, []string{DataScopeAll}, info.Branches)
	assert.Equal(t, int64(13), total, "应为全量 13 行")
	assert.Equal(t, sqlCount(t, ""), total)
}

// TestDataScopeM3ReverseOrder 反向顺序（ALL 在前）同样 applied=false，
// 证明 M3 的语义与角色顺序无关地收敛到"全部"。
func TestDataScopeM3ReverseOrder(t *testing.T) {
	lu := &entity.LoginUser{
		UserID: uCustomAll,
		DeptID: ptr(int64(106)),
		User: &entity.SysUser{
			UserID: uCustomAll,
			DeptID: ptr(int64(106)),
			Roles: []entity.SysRole{
				{RoleID: rAllAfter, RoleKey: "all_after", DataScope: DataScopeAll, Status: config.Normal, Permissions: []string{"system:user:list"}},
				{RoleID: rCustomC, RoleKey: "custom_c", DataScope: DataScopeCustom, Status: config.Normal, Permissions: []string{"system:user:list"}},
			},
		},
	}
	total, info := countUsers(t, lu, "system:user:list")
	assert.False(t, info.Applied)
	assert.Equal(t, int64(13), total)
}

// ---------------------------------------------------------------- 其余

// TestDataScopeParameterized 断言生成的数据权限条件**全部参数化**，
// 且不含任何由外部输入拼进 SQL 的字面量（AGENTS.md 全局规则 4）。
func TestDataScopeParameterized(t *testing.T) {
	cases := []struct {
		name     string
		userID   int64
		literals []string
	}{
		{"分支2 role_id", uRy, []string{"100", "101", "105"}},
		{"分支3 dept_id", uDept, []string{"105"}},
		{"分支4 dept_id", uDeptChild, []string{"101"}},
		{"分支5 user_id", uSelf, []string{"103"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lu := loadLoginUser(t, tc.userID)
			sqlStr, vars := dryRunSQL(t, lu, "system:user:list", "d", "u")
			t.Logf("SQL=%s VARS=%v", sqlStr, vars)
			assert.Contains(t, sqlStr, "?", "条件必须使用 ? 占位符")
			assert.NotEmpty(t, vars, "必须带绑定参数")
			for _, lit := range tc.literals {
				assert.NotContains(t, sqlStr, lit,
					"SQL 中出现字面量 %q，说明存在字符串拼接而非参数化", lit)
			}
		})
	}
}

// TestDataScopeSelfWithoutUserAlias 分支 5 在无 userAlias 时退化为 dept_id=0
// （DataScopeAspect.java:116-126）。
func TestDataScopeSelfWithoutUserAlias(t *testing.T) {
	lu := loadLoginUser(t, uSelf)
	scope, info := DataScopeWithInfo("system:user:list", lu, "d", "")
	assert.True(t, info.Applied)
	assert.False(t, info.Fallback, "这是分支 5 的正常退化，不是兜底")
	assert.Equal(t, []string{DataScopeSelf}, info.Branches)

	_, total, err := repository.SelectUserList(&query.UserList{},
		&page.PageDomain{PageNum: 1, PageSize: 1000}, scope)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total, "无 userAlias 的分支 5 应查不到任何数据（d.dept_id = 0）")
}

// TestDataScopePermissionEmpty 权限串为空时不按权限过滤角色
// （DataScopeAspect.java:86 的 isNotEmpty(permission) 短路）。
//
// 这条正是"SysRole.Permissions 未装载"与"装载了"的分水岭：
// ds_noperm 在 permission="" 时正常走分支 2，permission 非空时才兜底。
func TestDataScopePermissionEmpty(t *testing.T) {
	lu := loadLoginUser(t, uNoPerm)
	scope, info := DataScopeWithInfo("", lu, "d", "u")
	assert.True(t, info.Applied)
	assert.False(t, info.Fallback, "permission 为空时不应再按权限过滤角色")
	assert.Equal(t, []string{DataScopeCustom}, info.Branches)

	_, total, err := repository.SelectUserList(&query.UserList{},
		&page.PageDomain{PageNum: 1, PageSize: 1000}, scope)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total, "role ds_noperm 在 sys_role_dept 无记录，自定义分支命中 0 行")
}

// TestDataScopeRolePermissionsLoaded 守住 1b 修掉的缺口：
// 登录链路必须把角色权限集装载到 SysRole.Permissions，
// 否则 DataScopeAspect.java:86 的 containsAny 恒为 false -> 静默兜底。
func TestDataScopeRolePermissionsLoaded(t *testing.T) {
	for _, id := range []int64{uRy, uAll, uDept, uDeptChild, uSelf, uMix, uTwoCustom, uCustomAll} {
		u, err := repository.SelectUserByID(id)
		require.NoError(t, err)
		require.NotNil(t, u)
		_, err = GetMenuPermission(u)
		require.NoError(t, err)
		require.NotEmpty(t, u.Roles, "userID=%d 应有角色", id)
		for i := range u.Roles {
			assert.NotEmpty(t, u.Roles[i].Permissions,
				"userID=%d 的角色 %s 未装载 permissions，会导致数据权限静默兜底", id, u.Roles[i].RoleKey)
		}
	}
}

// TestDataScopeUnknownBranch 未知 data_scope 不产生条件，最终落到兜底。
func TestDataScopeUnknownBranch(t *testing.T) {
	lu := &entity.LoginUser{
		UserID: 999,
		User: &entity.SysUser{
			UserID: 999,
			DeptID: ptr(int64(105)),
			Roles:  []entity.SysRole{{RoleID: 9, RoleKey: "weird", DataScope: "9", Status: config.Normal}},
		},
	}
	_, info := DataScopeWithInfo("", lu, "d", "u")
	assert.True(t, info.Fallback, "未知 data_scope 不产生条件 -> conditions 为空 -> 兜底")
	assert.Equal(t, "无角色命中数据范围", info.Reason)
}

// TestFixturesNotCommitted 零副作用护栏：事务内的夹具不得出现在主库。
//
// 用事务外的连接（origDB）查询——REPEATABLE READ 下看不到未提交数据，
// 因此这里断言到的行数就是主库真实状态。
func TestFixturesNotCommitted(t *testing.T) {
	// sys_role_menu / sys_user_role / sys_role_dept 无 create_by 列（DDL 中只有两个 id 列），
	// 只对 sys_user / sys_role 做标记断言；关联表随事务回滚，一并消失。
	for _, tbl := range []string{"sys_user", "sys_role"} {
		var n int64
		require.NoError(t, origDB.Raw(fmt.Sprintf(
			"SELECT COUNT(*) FROM %s WHERE create_by = '1b-fixture'", tbl)).Scan(&n).Error)
		assert.Zero(t, n, "%s 里出现了夹具数据，测试不再零副作用", tbl)
	}
	var users, roles int64
	require.NoError(t, origDB.Raw("SELECT COUNT(*) FROM sys_user WHERE del_flag = '0'").Scan(&users).Error)
	require.NoError(t, origDB.Raw("SELECT COUNT(*) FROM sys_role WHERE del_flag = '0'").Scan(&roles).Error)
	assert.Equal(t, int64(2), users, "主库应保持种子数据的 2 个账号")
	assert.Equal(t, int64(2), roles, "主库应保持种子数据的 2 个角色")
}
