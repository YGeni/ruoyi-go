package service

import (
	"errors"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"

	"github.com/gin-gonic/gin"
)

// SelectRoleList 角色列表，等价
//
//	@DataScope(deptAlias = "d")
//	public List<SysRole> selectRoleList(SysRole role)
//
// **DataScope 五个落点之一**（deptAlias="d"、userAlias 取注解默认空串）。
// 因此数据范围 5（仅本人）在角色列表里退化成 d.dept_id = 0
// 与用户列表的 u.user_id = ? 语义不同 —— 单测 TestRoleDataScopeGolden 会分别断言。
func SelectRoleList(c *gin.Context, lu *entity.LoginUser, q *query.RoleList, p *page.PageDomain) ([]entity.SysRole, int64, error) {
	return repository.SelectRoleList(q, p, DataScope(c, lu, "d", ""))
}

// SelectAllocatedList 已分配该角色的用户，等价
//
//	@DataScope(deptAlias = "d", userAlias = "u")
//	public List<SysUser> selectAllocatedList(SysUser user)
//
// **DataScope 五个落点之一**。别名组合与 selectUserList 相同（d/u），
// 但基础查询多两个 join（sys_user_role / sys_role）且是 select distinct。
func SelectAllocatedList(c *gin.Context, lu *entity.LoginUser, roleID int64, q *query.AuthUserList, p *page.PageDomain) ([]entity.SysUser, int64, error) {
	return repository.SelectAllocatedList(roleID, q, p, DataScope(c, lu, "d", "u"))
}

// SelectUnallocatedList 未分配该角色的用户，等价
//
//	@DataScope(deptAlias = "d", userAlias = "u")
//	public List<SysUser> selectUnallocatedList(SysUser user)
//
// **DataScope 五个落点之一**。
func SelectUnallocatedList(c *gin.Context, lu *entity.LoginUser, roleID int64, q *query.AuthUserList, p *page.PageDomain) ([]entity.SysUser, int64, error) {
	return repository.SelectUnallocatedList(roleID, q, p, DataScope(c, lu, "d", "u"))
}

// SelectRoleAll 全部角色（optionselect 用），等价 SysRoleServiceImpl.selectRoleAll。
func SelectRoleAll() ([]entity.SysRole, error) {
	return repository.SelectRoleAll()
}

// SelectRoleByID 按主键查角色，等价 SysRoleServiceImpl.selectRoleById。
func SelectRoleByID(roleID int64) (*entity.SysRole, error) {
	return repository.SelectRoleByID(roleID)
}

// InsertRole 新增角色，等价 SysRoleServiceImpl.insertRole(L170-193).
//
// 校验顺序与 Java 一致：角色名唯一 -> 权限字符唯一 -> 插入 -> 写角色菜单关联。
func InsertRole(r *entity.SysRole, menuIDs []int64) error {
	ok, err := repository.CheckRoleNameUnique(r.RoleName, 0)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("新增角色'" + r.RoleName + "'失败，角色名称已存在")
	}
	ok, err = repository.CheckRoleKeyUnique(r.RoleKey, 0)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("新增角色'" + r.RoleName + "'失败，角色权限已存在")
	}
	if err := repository.InsertRole(r); err != nil {
		return err
	}
	return InsertRoleMenu(r.RoleID, menuIDs)
}

// InsertRoleMenu 写角色菜单关联，等价 SysRoleServiceImpl.insertRoleMenu(L246-254)。
//
// 幂等：先删后插。Java 侧是 deleteRoleMenu 再 batchRoleMenu 循环插入。
func InsertRoleMenu(roleID int64, menuIDs []int64) error {
	if err := repository.DeleteRoleMenu(roleID); err != nil {
		return err
	}
	return repository.InsertRoleMenus(roleID, menuIDs)
}

// UpdateRole 修改角色，等价 SysRoleServiceImpl.updateRole(L200-224)。
//
// Java 在更新时会带上 menu_check_strictly / dept_check_strictly（bool，零值有意义），
// Go 侧单独调用 UpdateRoleStrictly 覆盖，避免被"零值不更新"规则吞掉。
func UpdateRole(r *entity.SysRole, menuIDs []int64, menuStrict, deptStrict bool) error {
	exist, err := repository.SelectRoleByID(r.RoleID)
	if err != nil {
		return err
	}
	if exist == nil {
		return errors.New("角色不存在")
	}
	ok, err := repository.CheckRoleNameUnique(r.RoleName, r.RoleID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("修改角色'" + r.RoleName + "'失败，角色名称已存在")
	}
	ok, err = repository.CheckRoleKeyUnique(r.RoleKey, r.RoleID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("修改角色'" + r.RoleName + "'失败，角色权限已存在")
	}
	if err := repository.UpdateRole(r); err != nil {
		return err
	}
	if err := repository.UpdateRoleStrictly(r.RoleID, menuStrict, deptStrict); err != nil {
		return err
	}
	if menuIDs != nil {
		return repository.DeleteRoleMenu(r.RoleID)
	}
	return nil
}

// UpdateRoleStatus 修改角色状态，等价 SysRoleServiceImpl.updateRoleStatus(L231-243)。
//
// checkRoleAllowed / checkRoleDataScope 在 handler 层调用（需要 LoginUser）。
func UpdateRoleStatus(roleID int64, status string) error {
	return repository.UpdateRoleStatus(roleID, status)
}

// UpdateRoleDataScope 修改数据范围，等价 SysRoleServiceImpl.authDataScope(L257-270)。
//
// Java 在更新 data_scope 之后会重写 sys_role_dept；Go 侧保持一致。
func UpdateRoleDataScope(roleID int64, dataScope string, deptIDs []int64) error {
	if err := repository.UpdateRoleDataScope(roleID, dataScope); err != nil {
		return err
	}
	if err := repository.DeleteRoleDept(roleID); err != nil {
		return err
	}
	return repository.InsertRoleDepts(roleID, deptIDs)
}

// DeleteRoleByIDs 删除角色，等价 SysRoleServiceImpl.deleteRoleByIds(L277-291)。
//
// 前置校验：角色已分配给用户时不允许删除（SysRoleServiceImpl.checkRoleAllowed 之外
// 的 deleteRoleByIds 里循环 countUserRoleByRoleId）。
func DeleteRoleByIDs(ids []int64) error {
	for _, id := range ids {
		n, err := repository.CountUserRoles(id)
		if err != nil {
			return err
		}
		if n > 0 {
			return errors.New("角色已分配，不能删除")
		}
	}
	return repository.DeleteRoleByIDs(ids)
}

// CheckRoleAllowed 校验是否允许操作该角色，等价 SysRoleServiceImpl.checkRoleAllowed(L293-304)。
//
// 规则：admin 角色（roleKey="admin" 或 roleId=1）不允许被增删改。
func CheckRoleAllowed(r *entity.SysRole) error {
	if r == nil {
		return nil
	}
	if r.IsAdmin() || r.RoleID == 1 {
		return errors.New("不允许操作超级管理员角色")
	}
	return nil
}

// CheckRoleDataScope 校验数据权限，等价 SysRoleServiceImpl.checkRoleDataScope(L306-313)。
//
// Java：非超管用户只能操作自己数据范围内的角色。
func CheckRoleDataScope(lu *entity.LoginUser, roleID int64) error {
	if lu == nil || lu.IsAdmin() {
		return nil
	}
	roles, err := repository.SelectRolesByUserID(lu.UserID)
	if err != nil {
		return err
	}
	for i := range roles {
		if roles[i].RoleID == roleID {
			return nil
		}
	}
	return errors.New("没有权限访问角色数据！")
}

// SelectRoleDeptTree 角色已授权部门 ID（/system/role/deptTree/{roleId}），
// 等价 SysDeptServiceImpl.selectDeptListByRoleId(L105-110)。
func SelectRoleDeptTree(roleID int64) ([]int64, error) {
	r, err := repository.SelectRoleByID(roleID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errors.New("角色不存在")
	}
	return repository.SelectDeptIDsByRoleID(roleID, r.DeptCheckStrictly)
}

// CancelAuthUser 取消单个用户的角色，等价 SysUserServiceImpl.deleteAuthUser(L354-361)。
func CancelAuthUser(userID, roleID int64) error {
	return repository.DeleteUserRoleByIDs(userID, roleID)
}

// CancelAllAuthUser 批量取消授权，等价 SysUserServiceImpl.deleteAuthUsers(L368-377)
// + SysUserRoleMapper.xml deleteUserRoleInfos（role_id + user_id in (...)）。
//
// Java 侧 userIds 为必填（前端 cancelAll 会带上整页选中项），此处保持一致。
func CancelAllAuthUser(roleID int64, userIDs []int64) error {
	if len(userIDs) == 0 {
		return errors.New("未选择要取消授权的用户")
	}
	for _, u := range userIDs {
		if err := repository.DeleteUserRoleByIDs(u, roleID); err != nil {
			return err
		}
	}
	return nil
}

// InsertAuthUsers 批量授权用户角色，等价 SysUserServiceImpl.insertAuthUsers(L384-393)。
//
// 与 selectAll 场景对应：Java 侧直接 batchUserRole 插入（不先删），
// 主键冲突会报错；Go 侧保持一致，由前端避免重复提交。
func InsertAuthUsers(roleID int64, userIDs []int64) error {
	if len(userIDs) == 0 {
		return errors.New("未选择要授权的用户")
	}
	return repository.InsertUserRolesForRole(roleID, userIDs)
}
