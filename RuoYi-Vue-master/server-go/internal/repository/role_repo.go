package repository

import (
	"errors"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"

	"gorm.io/gorm"
)

// roleListQuery 构造 selectRoleList 的基础查询（SysRoleMapper.xml:25-61）。
//
// Java 的 selectRoleVo 是 select distinct + 三表 left join：
//
//	from sys_role r
//	left join sys_user_role ur on ur.role_id = r.role_id
//	left join sys_user u on u.user_id = ur.user_id
//	left join sys_dept d on u.dept_id = d.dept_id
//	where r.del_flag = '0' ... ${params.dataScope} order by r.role_sort
//
// 别名 r/u/d 是硬约束：的 @DataScope(deptAlias = "d")
// 生成的条件直接引用 d.dept_id。
//
// **注意 userAlias 为空**：@DataScope(deptAlias="d") 只写了 deptAlias，
// userAlias 取注解默认值 ""，因此数据范围 5（仅本人）
// 在这里会退化成 d.dept_id = 0。
func roleListQuery(q *query.RoleList, scope func(*gorm.DB) *gorm.DB) *gorm.DB {
	tx := db.DB.Table("sys_role r").
		Select("DISTINCT r.*").
		Joins("LEFT JOIN sys_user_role ur ON ur.role_id = r.role_id").
		Joins("LEFT JOIN sys_user u ON u.user_id = ur.user_id").
		Joins("LEFT JOIN sys_dept d ON u.dept_id = d.dept_id").
		Where("r.del_flag = ?", "0")
	if q.RoleID != 0 {
		tx = tx.Where("r.role_id = ?", q.RoleID)
	}
	if q.RoleName != "" {
		tx = tx.Where("r.role_name LIKE ?", "%"+q.RoleName+"%")
	}
	if q.RoleKey != "" {
		tx = tx.Where("r.role_key LIKE ?", "%"+q.RoleKey+"%")
	}
	if q.Status != "" {
		tx = tx.Where("r.status = ?", q.Status)
	}
	if q.BeginTime != "" {
		tx = tx.Where("date_format(r.create_time,'%Y%m%d') >= date_format(?,'%Y%m%d')", q.BeginTime)
	}
	if q.EndTime != "" {
		tx = tx.Where("date_format(r.create_time,'%Y%m%d') <= date_format(?,'%Y%m%d')", q.EndTime)
	}
	if scope != nil {
		tx = tx.Scopes(scope)
	}
	return tx
}

// SelectRoleList 分页查询角色，等价
// @DataScope(deptAlias = "d") + SysRoleMapper.xml:36-61。
//
// 排序固定 r.role_sort（Java 写死在 SQL 末尾，不接受前端 orderBy 覆盖），
// 由 page.Paginate 的 defaultOrder 传入。
func SelectRoleList(q *query.RoleList, p *page.PageDomain, scope func(*gorm.DB) *gorm.DB) ([]entity.SysRole, int64, error) {
	var roles []entity.SysRole
	tx := roleListQuery(q, scope).Order("r.role_sort")
	// distinct 场景的计数必须用 DISTINCT 主键，否则 left join 放大行数
	total, err := page.Paginate(tx, p, "r", "DISTINCT r.role_id", &roles)
	if err != nil {
		return nil, 0, err
	}
	return roles, total, nil
}

// SelectRoleAll 查询全部角色（不分页），等价 SysRoleMapper.xml selectRoleAll。
func SelectRoleAll() ([]entity.SysRole, error) {
	var roles []entity.SysRole
	err := db.DB.Where("del_flag = ?", "0").Order("role_sort").Find(&roles).Error
	return roles, err
}

// SelectRoleByID 按主键查角色，等价 SysRoleMapper.xml selectRoleById。
func SelectRoleByID(roleID int64) (*entity.SysRole, error) {
	var r entity.SysRole
	err := db.DB.Where("role_id = ?", roleID).First(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// SelectRolesByUserID 查用户已绑定的角色，等价 SysRoleMapper.xml selectRolePermissionByUserId。
func SelectRolesByUserID(userID int64) ([]entity.SysRole, error) {
	var roles []entity.SysRole
	err := db.DB.Table("sys_role r").
		Select("r.*").
		Joins("LEFT JOIN sys_user_role ur ON ur.role_id = r.role_id").
		Joins("LEFT JOIN sys_user u ON u.user_id = ur.user_id").
		Joins("LEFT JOIN sys_dept d ON u.dept_id = d.dept_id").
		Where("r.del_flag = ?", "0").
		Where("ur.user_id = ?", userID).
		Find(&roles).Error
	return roles, err
}

// CheckRoleNameUnique 角色名是否可用，等价 SysRoleMapper.xml checkRoleNameUnique。
func CheckRoleNameUnique(name string, excludeRoleID int64) (bool, error) {
	return roleNotExists("role_name", name, excludeRoleID)
}

// CheckRoleKeyUnique 角色权限字符是否可用，等价 SysRoleMapper.xml checkRoleKeyUnique。
func CheckRoleKeyUnique(key string, excludeRoleID int64) (bool, error) {
	return roleNotExists("role_key", key, excludeRoleID)
}

func roleNotExists(col, val string, excludeRoleID int64) (bool, error) {
	tx := db.DB.Table("sys_role").Where(col+" = ?", val).Where("del_flag = ?", "0")
	if excludeRoleID != 0 {
		tx = tx.Where("role_id != ?", excludeRoleID)
	}
	var n int64
	if err := tx.Count(&n).Error; err != nil {
		return false, err
	}
	return n == 0, nil
}

// InsertRole 新增角色，等价 SysRoleMapper.xml insertRole（动态列）。
//
// DIFF(review): Java 的 insertRole 不含 del_flag，靠 DDL 默认值 '0' 填充；
// Go 的 Create 会**显式写入零值**导致 del_flag=”，列表随即查不到新角色。
// 这里显式补默认值，与 DB 默认对齐（status 同理）。
func InsertRole(r *entity.SysRole) error {
	if r.DelFlag == "" {
		r.DelFlag = model.DelFlagExists
	}
	if r.Status == "" {
		r.Status = config.Normal
	}
	return db.DB.Create(r).Error
}

// UpdateRole 修改角色，等价 SysRoleMapper.xml updateRole 的 <set> 动态列。
//
// 显式列清单与 Java 的 if 分支一一对应：role_name / role_key / role_sort /
// data_scope / menu_check_strictly / dept_check_strictly / status / remark / update_by。
// 不更新 role_id 与 del_flag。
func UpdateRole(r *entity.SysRole) error {
	sets := map[string]any{"update_time": time.Now()}
	if r.RoleName != "" {
		sets["role_name"] = r.RoleName
	}
	if r.RoleKey != "" {
		sets["role_key"] = r.RoleKey
	}
	if r.RoleSort != 0 {
		sets["role_sort"] = r.RoleSort
	}
	if r.DataScope != "" {
		sets["data_scope"] = r.DataScope
	}
	if r.Status != "" {
		sets["status"] = r.Status
	}
	if r.Remark != "" {
		sets["remark"] = r.Remark
	}
	if r.UpdateBy != "" {
		sets["update_by"] = r.UpdateBy
	}
	// menu_check_strictly / dept_check_strictly 是 bool 且合法值含 false，
	// 无法用零值判断"是否传入"，故由调用方（service）按需覆盖。
	return db.DB.Table("sys_role").Where("role_id = ?", r.RoleID).Updates(sets).Error
}

// UpdateRoleStrictly 单独更新两个 xxx_check_strictly 开关（bool 无法用零值判断）。
func UpdateRoleStrictly(roleID int64, menuStrict, deptStrict bool) error {
	return db.DB.Table("sys_role").Where("role_id = ?", roleID).
		Updates(map[string]any{
			"menu_check_strictly": menuStrict,
			"dept_check_strictly": deptStrict,
			"update_time":         time.Now(),
		}).Error
}

// UpdateRoleStatus 只改状态，等价 SysRoleMapper.xml 的 updateRoleStatus。
func UpdateRoleStatus(roleID int64, status string) error {
	return db.DB.Table("sys_role").Where("role_id = ?", roleID).
		Updates(map[string]any{"status": status, "update_time": time.Now()}).Error
}

// UpdateRoleDataScope 修改数据范围，等价 SysRoleServiceImpl.insertRoleAuth 的数据范围部分。
func UpdateRoleDataScope(roleID int64, dataScope string) error {
	return db.DB.Table("sys_role").Where("role_id = ?", roleID).
		Updates(map[string]any{"data_scope": dataScope, "update_time": time.Now()}).Error
}

// DeleteRoleByIDs 逻辑删除角色，等价 SysRoleMapper.xml deleteRoleByIds：del_flag = '2'。
func DeleteRoleByIDs(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return db.DB.Table("sys_role").Where("role_id IN ?", ids).
		Updates(map[string]any{"del_flag": "2", "update_time": time.Now()}).Error
}

// DeleteRoleMenu 删除角色的菜单关联，等价 SysRoleMapper.xml deleteRoleMenu。
func DeleteRoleMenu(roleID int64) error {
	return db.DB.Where("role_id = ?", roleID).Delete(&entity.SysRoleMenu{}).Error
}

// InsertRoleMenus 批量写入角色菜单关联，等价 SysRoleMapper.xml batchRoleMenu。
func InsertRoleMenus(roleID int64, menuIDs []int64) error {
	if len(menuIDs) == 0 {
		return nil
	}
	rows := make([]entity.SysRoleMenu, 0, len(menuIDs))
	for _, m := range menuIDs {
		rows = append(rows, entity.SysRoleMenu{RoleID: roleID, MenuID: m})
	}
	return db.DB.Create(&rows).Error
}

// DeleteRoleDept 删除角色部门关联，等价 SysRoleMapper.xml deleteRoleDept。
func DeleteRoleDept(roleID int64) error {
	return db.DB.Where("role_id = ?", roleID).Delete(&entity.SysRoleDept{}).Error
}

// InsertRoleDepts 批量写入角色部门关联，等价 SysRoleMapper.xml batchRoleDept。
func InsertRoleDepts(roleID int64, deptIDs []int64) error {
	if len(deptIDs) == 0 {
		return nil
	}
	rows := make([]entity.SysRoleDept, 0, len(deptIDs))
	for _, d := range deptIDs {
		rows = append(rows, entity.SysRoleDept{RoleID: roleID, DeptID: d})
	}
	return db.DB.Create(&rows).Error
}

// SelectDeptIDsByRoleID 查角色已授权的部门，等价 SysDeptMapper.xml selectDeptListByRoleId。
//
// deptCheckStrictly=true 时排除"父部门也在结果集内"的行（Java 的 not in 子查询）。
func SelectDeptIDsByRoleID(roleID int64, deptCheckStrictly bool) ([]int64, error) {
	tx := db.DB.Table("sys_dept d").
		Select("d.dept_id").
		Joins("LEFT JOIN sys_role_dept rd ON d.dept_id = rd.dept_id").
		Where("rd.role_id = ?", roleID)
	if deptCheckStrictly {
		tx = tx.Where("d.dept_id NOT IN (SELECT d.parent_id FROM sys_dept d "+
			"INNER JOIN sys_role_dept rd ON d.dept_id = rd.dept_id AND rd.role_id = ?)", roleID)
	}
	tx = tx.Order("d.parent_id, d.order_num")
	var ids []int64
	err := tx.Pluck("d.dept_id", &ids).Error
	return ids, err
}

// DeleteUserRoleByIDs 取消单个用户的角色，等价 SysUserRoleMapper.xml deleteUserRoleInfo。
func DeleteUserRoleByIDs(userID, roleID int64) error {
	return db.DB.Where("user_id = ?", userID).Where("role_id = ?", roleID).
		Delete(&entity.SysUserRole{}).Error
}

// InsertUserRolesForRole 批量插入「同一角色 + 多个用户」的关联，
// 等价 SysUserRoleMapper.xml batchUserRole（selectAll 场景）。
//
// DIFF(review): user_repo.go 的 InsertUserRoles 是「同一用户 + 多个角色」，
// 方向相反，这里单独提供。
func InsertUserRolesForRole(roleID int64, userIDs []int64) error {
	if len(userIDs) == 0 {
		return nil
	}
	rows := make([]entity.SysUserRole, 0, len(userIDs))
	for _, u := range userIDs {
		rows = append(rows, entity.SysUserRole{UserID: u, RoleID: roleID})
	}
	return db.DB.Create(&rows).Error
}

// CountUserRoles 统计角色已关联的用户数，等价 SysUserRoleMapper.xml countUserRoleByRoleId
// （用于 deleteRoleByIds 前的"已分配不能删除"校验）。
func CountUserRoles(roleID int64) (int64, error) {
	var n int64
	err := db.DB.Table("sys_user_role").Where("role_id = ?", roleID).Count(&n).Error
	return n, err
}
