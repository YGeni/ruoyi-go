package repository

import (
	"errors"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"

	"gorm.io/gorm"
)

// 菜单可见范围：只取目录与菜单（M/C），与原 SQL 的 menu_type in ('M','C') 一致。
var menuTypes = []string{config.TypeDir, config.TypeMenu}

// SelectMenuTreeAll 查询全部菜单，等价于 SysMenuMapper.xml:52-56 的 selectMenuTreeAll。
//
// 原 SQL：where m.menu_type in ('M','C') and m.status = 0 order by m.parent_id, m.order_num
// DIFF(review): status 列为 char(1)，Java 用 m.status = 0（MySQL 隐式转换）；
// Go 侧按 char(1) 用 '0' 精确匹配，语义等价且不受隐式转换影响。
func SelectMenuTreeAll() ([]entity.SysMenu, error) {
	var menus []entity.SysMenu
	err := db.DB.Where("menu_type IN ?", menuTypes).
		Where("status = ?", config.Normal).
		Order("parent_id").Order("order_num").
		Find(&menus).Error
	return menus, err
}

// SelectMenuTreeByUserID 按用户查询菜单，等价于 SysMenuMapper.xml:77-86 的
// selectMenuTreeByUserId（四表 left join + distinct）。
func SelectMenuTreeByUserID(userID int64) ([]entity.SysMenu, error) {
	var menus []entity.SysMenu
	err := db.DB.Table("sys_menu m").
		Select("DISTINCT m.*").
		Joins("LEFT JOIN sys_role_menu rm ON m.menu_id = rm.menu_id").
		Joins("LEFT JOIN sys_user_role ur ON rm.role_id = ur.role_id").
		Joins("LEFT JOIN sys_role ro ON ur.role_id = ro.role_id").
		Joins("LEFT JOIN sys_user u ON ur.user_id = u.user_id").
		Where("u.user_id = ?", userID).
		Where("m.menu_type IN ?", menuTypes).
		Where("m.status = ?", config.Normal).
		Where("ro.status = ?", config.Normal).
		Order("m.parent_id").Order("m.order_num").
		Scan(&menus).Error
	if err != nil {
		return nil, err
	}
	return menus, nil
}

// SelectMenuPermsByUserID 查询用户权限串，等价于 SysMenuMapper.xml:106-113。
//
// 返回值可能含逗号分隔的多个权限（如 "system:user:add,system:user:edit"），
// 由调用方按 Constants.PERMISSION_DELIMITER 拆分，与 SysMenuServiceImpl:100-108 一致。
func SelectMenuPermsByUserID(userID int64) ([]string, error) {
	var perms []string
	err := db.DB.Table("sys_menu m").
		Distinct().
		Joins("LEFT JOIN sys_role_menu rm ON m.menu_id = rm.menu_id").
		Joins("LEFT JOIN sys_user_role ur ON rm.role_id = ur.role_id").
		Joins("LEFT JOIN sys_role r ON r.role_id = ur.role_id").
		Where("m.status = ?", config.Normal).
		Where("r.status = ?", config.Normal).
		Where("ur.user_id = ?", userID).
		Pluck("m.perms", &perms).Error
	return perms, err
}

// SelectMenuPermsByRoleID 查询角色权限串，等价于 SysMenuMapper.xml:115-120。
func SelectMenuPermsByRoleID(roleID int64) ([]string, error) {
	var perms []string
	err := db.DB.Table("sys_menu m").
		Distinct().
		Joins("LEFT JOIN sys_role_menu rm ON m.menu_id = rm.menu_id").
		Where("m.status = ?", config.Normal).
		Where("rm.role_id = ?", roleID).
		Pluck("m.perms", &perms).Error
	return perms, err
}

// SelectRoleKeysByUserID 查询用户角色标识，等价于 SysRoleMapper.xml:59-62 的
// selectRolePermissionByUserId（where r.del_flag = '0' and ur.user_id = #{userId}）。
func SelectRoleKeysByUserID(userID int64) ([]string, error) {
	var keys []string
	err := db.DB.Table("sys_role r").
		Distinct().
		Joins("LEFT JOIN sys_user_role ur ON ur.role_id = r.role_id").
		Where("r.del_flag = ?", model.DelFlagExists).
		Where("ur.user_id = ?", userID).
		Pluck("r.role_key", &keys).Error
	return keys, err
}

// ---------- 菜单管理 CRUD（SysMenuMapper.xml）----------

// applyMenuFilters 应用 selectMenuList / selectMenuListByUserId 共有的三个动态条件
// （SysMenuMapper.xml:39-47：menu_name like、visible、status）。
//
// alias 用于带 join 的查询：sys_role 也有 status 列，
// 不写成 "m.status" 会触发 MySQL 1052 Column 'status' in where clause is ambiguous。
func applyMenuFilters(tx *gorm.DB, q *query.MenuList, alias string) *gorm.DB {
	if q == nil {
		return tx
	}
	if q.MenuName != "" {
		tx = tx.Where(alias+"menu_name LIKE ?", "%"+q.MenuName+"%")
	}
	if q.Visible != "" {
		tx = tx.Where(alias+"visible = ?", q.Visible)
	}
	if q.Status != "" {
		tx = tx.Where(alias+"status = ?", q.Status)
	}
	return tx
}

// SelectMenuList 查询菜单列表，等价 SysMenuMapper.xml:36-50 的 selectMenuList。
//
// DIFF(review): 原 SQL **没有 ORDER BY**（前端自己 handleTree 排序），此处保持无排序。
func SelectMenuList(q *query.MenuList) ([]entity.SysMenu, error) {
	var menus []entity.SysMenu
	tx := applyMenuFilters(db.DB.Model(&entity.SysMenu{}), q, "")
	err := tx.Find(&menus).Error
	if err != nil {
		return nil, err
	}
	return menus, nil
}

// SelectMenuListByUserID 按用户查询菜单，等价 SysMenuMapper.xml:58-76 的
// selectMenuListByUserId（三表 left join + distinct + 同样三个动态条件）。
func SelectMenuListByUserID(q *query.MenuList, userID int64) ([]entity.SysMenu, error) {
	var menus []entity.SysMenu
	tx := db.DB.Table("sys_menu m").
		Select("DISTINCT m.*").
		Joins("LEFT JOIN sys_role_menu rm ON m.menu_id = rm.menu_id").
		Joins("LEFT JOIN sys_user_role ur ON rm.role_id = ur.role_id").
		Joins("LEFT JOIN sys_role ro ON ur.role_id = ro.role_id").
		Where("ur.user_id = ?", userID)
	tx = applyMenuFilters(tx, q, "m.")
	err := tx.Scan(&menus).Error
	if err != nil {
		return nil, err
	}
	return menus, nil
}

// SelectMenuByID 按主键查菜单，等价 SysMenuMapper.xml:122-125。
func SelectMenuByID(menuID int64) (*entity.SysMenu, error) {
	var m entity.SysMenu
	err := db.DB.Where("menu_id = ?", menuID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// CheckMenuNameUnique 同父菜单下菜单名是否重复，等价 SysMenuMapper.xml:131-134。
// 返回命中的那条记录（含 menu_id），由调用方排除自身；无命中返回 nil。
func CheckMenuNameUnique(menuName string, parentID int64) (*entity.SysMenu, error) {
	var m entity.SysMenu
	err := db.DB.Where("menu_name = ?", menuName).
		Where("parent_id = ?", parentID).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// SelectMenusByPathOrRouteName 按 path 或 routeName 查菜单，
// 等价 SysMenuMapper.xml:136-138（menu_type in ('M','C') + 四列 OR）。
func SelectMenusByPathOrRouteName(path, routeName string) ([]entity.SysMenu, error) {
	var menus []entity.SysMenu
	err := db.DB.Where("menu_type IN ?", menuTypes).
		Where("path = ? OR path = ? OR route_name = ? OR route_name = ?",
			path, routeName, path, routeName).
		Find(&menus).Error
	if err != nil {
		return nil, err
	}
	return menus, nil
}

// HasChildByMenuID 是否存在子菜单，等价 SysMenuMapper.xml:127-129。
func HasChildByMenuID(menuID int64) (bool, error) {
	var n int64
	err := db.DB.Model(&entity.SysMenu{}).Where("parent_id = ?", menuID).Count(&n).Error
	return n > 0, err
}

// CheckMenuExistRole 菜单是否已分配给角色，等价 SysRoleMenuMapper.xml 的 checkMenuExistRole。
func CheckMenuExistRole(menuID int64) (bool, error) {
	var n int64
	err := db.DB.Table("sys_role_menu").Where("menu_id = ?", menuID).Count(&n).Error
	return n > 0, err
}

// InsertMenu 新增菜单，等价 SysMenuMapper.xml:165-196 的 insertMenu（动态列）。
//
// **本库 sys_menu 的 is_frame / is_cache 是 int，不是若依标准的 char(1)**
// （DESC sys_menu 实测；visible / status / menu_type 仍是 char(1)）。
// 实体按 Java 侧建模为 string，直接 Create 会把空串写进 int 列，报
// Error 1366 "Incorrect integer value: ” for column 'is_cache'"。
// 因此这两个字段为空时必须 Omit（由 DDL 默认值 1 / 0 填充），
// 有值时 MySQL 会对 "0"/"1" 做隐式转换，与 char(1) 行为一致。
//
// menu_id 未指定时同样 Omit，交给自增主键。
func InsertMenu(m *entity.SysMenu) error {
	var omit []string
	if m.MenuID == 0 {
		omit = append(omit, "menu_id")
	}
	if m.IsFrame == "" {
		omit = append(omit, "is_frame")
	}
	if m.IsCache == "" {
		omit = append(omit, "is_cache")
	}
	tx := db.DB
	if len(omit) > 0 {
		tx = tx.Omit(omit...)
	}
	return tx.Create(m).Error
}

// UpdateMenu 修改菜单，等价 SysMenuMapper.xml:141-163 的 updateMenu（动态列 + update_time = sysdate()）。
//
// 只写 Java 侧 `<if test=...>` 判定为真的列：
// 空字符串一律跳过，避免把 char(1) 的 visible/status 等写成空串
// （Go 零值与 Java null 不等价，照搬零值会清掉状态字段，见 AGENTS.md）。
func UpdateMenu(m *entity.SysMenu) error {
	upd := map[string]any{
		"parent_id":   m.ParentID,
		"order_num":   m.OrderNum,
		"update_time": time.Now(),
	}
	if m.MenuName != "" {
		upd["menu_name"] = m.MenuName
	}
	if m.Path != "" {
		upd["path"] = m.Path
	}
	if m.Component != "" {
		upd["component"] = m.Component
	}
	if m.Query != "" {
		upd["query"] = m.Query
	}
	if m.RouteName != "" {
		upd["route_name"] = m.RouteName
	}
	if m.IsFrame != "" {
		upd["is_frame"] = m.IsFrame
	}
	if m.IsCache != "" {
		upd["is_cache"] = m.IsCache
	}
	if m.MenuType != "" {
		upd["menu_type"] = m.MenuType
	}
	if m.Visible != "" {
		upd["visible"] = m.Visible
	}
	if m.Status != "" {
		upd["status"] = m.Status
	}
	if m.Perms != "" {
		upd["perms"] = m.Perms
	}
	if m.Icon != "" {
		upd["icon"] = m.Icon
	}
	if m.Remark != "" {
		upd["remark"] = m.Remark
	}
	if m.UpdateBy != "" {
		upd["update_by"] = m.UpdateBy
	}
	return db.DB.Model(&entity.SysMenu{}).Where("menu_id = ?", m.MenuID).Updates(upd).Error
}

// UpdateMenuSort 只改排序号，等价 SysMenuMapper.xml 的 updateMenuSort。
func UpdateMenuSort(menuID int64, orderNum int) error {
	return db.DB.Model(&entity.SysMenu{}).Where("menu_id = ?", menuID).
		Updates(map[string]any{"order_num": orderNum, "update_time": time.Now()}).Error
}

// DeleteMenuByID 删除菜单，等价 SysMenuMapper.xml 的 deleteMenuById。
func DeleteMenuByID(menuID int64) error {
	return db.DB.Where("menu_id = ?", menuID).Delete(&entity.SysMenu{}).Error
}

// SelectMenuIDsByRoleID 查询角色已分配的菜单 ID，
// 等价 SysMenuMapper.xml:88-97 的 selectMenuListByRoleId。
//
// menuCheckStrictly（sys_role.menu_check_strictly）为真时排除掉"其父菜单也在结果集里"的子菜单，
// 即只保留叶子 —— Java 的 SQL 是
// `and m.menu_id not in (select m.parent_id from sys_menu m inner join sys_role_menu rm ... and rm.role_id = ?)`。
func SelectMenuIDsByRoleID(roleID int64, menuCheckStrictly bool) ([]int64, error) {
	var ids []int64
	tx := db.DB.Table("sys_menu m").
		Joins("LEFT JOIN sys_role_menu rm ON m.menu_id = rm.menu_id").
		Where("rm.role_id = ?", roleID)
	if menuCheckStrictly {
		tx = tx.Where(`m.menu_id NOT IN (SELECT m.parent_id FROM sys_menu m
			INNER JOIN sys_role_menu rm ON m.menu_id = rm.menu_id AND rm.role_id = ?)`, roleID)
	}
	err := tx.Order("m.parent_id").Order("m.order_num").Pluck("m.menu_id", &ids).Error
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []int64{}
	}
	return ids, nil
}
