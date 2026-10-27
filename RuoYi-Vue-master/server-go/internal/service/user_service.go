package service

import (
	"strconv"
	"strings"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
)

// GetRolePermission 等价于 SysPermissionService.getRolePermission(L37-50)：
// 管理员返回 {"admin"}，否则返回 roleKey 集合。
func GetRolePermission(user *entity.SysUser) ([]string, error) {
	if user.IsAdmin() {
		return []string{config.SuperAdmin}, nil
	}
	return repository.SelectRoleKeysByUserID(user.UserID)
}

// GetMenuPermission 等价于 SysPermissionService.getMenuPermission(L58-88)：
//   - 管理员 -> {"*:*:*"}
//   - 否则遍历角色（跳过停用角色与 admin 角色），合并每个角色的菜单权限
//   - 无任何角色时按 userId 直接查菜单权限
//
// 每个角色的权限串可能形如 "a:b:c,d:e:f"，需按逗号拆分（SysMenuServiceImpl:100-108）。
func GetMenuPermission(user *entity.SysUser) ([]string, error) {
	if user.IsAdmin() {
		return []string{config.AllPermission}, nil
	}
	set := map[string]struct{}{}
	if len(user.Roles) > 0 {
		for _, role := range user.Roles {
			if role.Status != config.Normal || role.IsAdmin() {
				continue
			}
			perms, err := repository.SelectMenuPermsByRoleID(role.RoleID)
			if err != nil {
				return nil, err
			}
			for _, p := range splitPerms(perms) {
				set[p] = struct{}{}
			}
		}
	} else {
		perms, err := repository.SelectMenuPermsByUserID(user.UserID)
		if err != nil {
			return nil, err
		}
		for _, p := range splitPerms(perms) {
			set[p] = struct{}{}
		}
	}
	res := make([]string, 0, len(set))
	for p := range set {
		res = append(res, p)
	}
	return res, nil
}

func splitPerms(perms []string) []string {
	out := make([]string, 0, len(perms))
	for _, perm := range perms {
		if strings.TrimSpace(perm) == "" {
			continue
		}
		for _, item := range strings.Split(perm, config.PermissionDelimiter) {
			if p := strings.TrimSpace(item); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

// AccountChrtype 对应 SysLoginController.getSysAccountChrtype(L110-113)：
// sys.account.chrtype，默认 "0"。
func AccountChrtype() string {
	v, ok, err := repository.SelectConfigValueByKey(config.ConfigChrtype)
	if err != nil || !ok || v == "" {
		return "0"
	}
	return v
}

// InitPasswordIsModify 对应 SysLoginController.initPasswordIsModify(L116-120)：
// sys.account.initPasswordModify == 1 且从未改过密码（pwd_update_date 为空）时返回 true。
func InitPasswordIsModify(pwdUpdateDate *time.Time) bool {
	v, ok, err := repository.SelectConfigValueByKey(config.ConfigInitPasswordModify)
	if err != nil || !ok {
		return false
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return false
	}
	return n == 1 && pwdUpdateDate == nil
}

// PasswordIsExpiration 对应 SysLoginController.passwordIsExpiration(L123-137)：
// sys.account.passwordValidateDays > 0 且超过天数即过期；从未改过密码直接视为过期。
func PasswordIsExpiration(pwdUpdateDate *time.Time) bool {
	v, ok, err := repository.SelectConfigValueByKey(config.ConfigPasswordValidDays)
	if err != nil || !ok {
		return false
	}
	days, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || days <= 0 {
		return false
	}
	if pwdUpdateDate == nil {
		return true
	}
	return time.Since(*pwdUpdateDate) > time.Duration(days)*24*time.Hour
}
