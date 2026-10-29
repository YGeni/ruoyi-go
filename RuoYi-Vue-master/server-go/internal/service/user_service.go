package service

import (
	"strconv"
	"strings"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"

	"github.com/gin-gonic/gin"
)

// SelectUserList 用户列表，等价于
//
//	@DataScope(deptAlias = "d", userAlias = "u")
//	public List<SysUser> selectUserList(SysUser user)
//
// 数据权限由 DataScope 以 GORM Scope 注入（Java 侧是 AOP 拼 ${params.dataScope}）。
func SelectUserList(c *gin.Context, lu *entity.LoginUser, q *query.UserList, p *page.PageDomain) ([]entity.SysUser, int64, error) {
	return repository.SelectUserList(q, p, DataScope(c, lu, "d", "u"))
}

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
//
// **关键副作用（1b 补齐）**：必须同时把每个角色的权限集回填到 role.Permissions
// 的 role.setPermissions(rolePerms)）。
// 的 containsAny(role.getPermissions, permission) 依赖它
// 若缺失，任何带 permission 的 @DataScope 调用都会判定"所有角色都未持有该权限"
// -> conditions 为空 -> 落到 "OR d.dept_id = 0" 兜底 -> 列表静默返回 0 行。
// 该副作用经由 LoginUser.User 一并写进 Redis login_tokens:<uuid>，
// 因此只需在登录（createToken）与 getInfo 时执行即可覆盖全部请求。
func GetMenuPermission(user *entity.SysUser) ([]string, error) {
	if user.IsAdmin() {
		return []string{config.AllPermission}, nil
	}
	set := map[string]struct{}{}
	if len(user.Roles) > 0 {
		for i := range user.Roles {
			role := &user.Roles[i]
			if role.Status != config.Normal || role.IsAdmin() {
				continue
			}
			perms, err := repository.SelectMenuPermsByRoleID(role.RoleID)
			if err != nil {
				return nil, err
			}
			// 见函数注释：@DataScope 的角色级权限匹配依赖此字段，不可省略。
			role.Permissions = perms
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
