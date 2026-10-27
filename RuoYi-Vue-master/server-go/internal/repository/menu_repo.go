package repository

import (
	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
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
