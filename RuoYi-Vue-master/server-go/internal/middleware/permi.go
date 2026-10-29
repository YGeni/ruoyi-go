package middleware

import (
	"net/http"
	"strings"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

// 本节替代 Java 的 @PreAuthorize("@ss.hasPermi(...)")，
// 对应 ruoyi-。
//
// Java 侧要点复刻：
//   - hasPermi(L27-40)：空权限直接 false；命中判定见 hasPermissions(L155-158)
//   - hasPermissions(L155-158)：permissions.contains("*:*:*") || contains(trim(permission))
//   - hasAnyPermi(L59-80)：按 "," 拆分，任一命中即 true
//   - hasRole(L88-108)：roleKey == "admin" 直通，否则按 trim 后的值比对
//   - hasAnyRoles(L127-146)：按 "," 拆分，任一命中即 true

// HasPermi 等价于 PermissionService.hasPermi：包含 "*:*:*" 或目标权限即放行。
func HasPermi(permissions []string, permission string) bool {
	permission = strings.TrimSpace(permission)
	if permission == "" {
		return false
	}
	for _, p := range permissions {
		if p == config.AllPermission || p == permission {
			return true
		}
	}
	return false
}

// HasAnyPermi 等价于 PermissionService.hasAnyPermi（分隔符 Constants.PERMISSION_DELIMITER）。
func HasAnyPermi(permissions []string, any string) bool {
	if strings.TrimSpace(any) == "" {
		return false
	}
	for _, p := range strings.Split(any, config.PermissionDelimiter) {
		if HasPermi(permissions, p) {
			return true
		}
	}
	return false
}

// HasRole 等价于 PermissionService.hasRole：roleKey=admin 直通。
func HasRole(roleKeys []string, role string) bool {
	role = strings.TrimSpace(role)
	if role == "" {
		return false
	}
	for _, k := range roleKeys {
		if k == config.SuperAdmin || k == role {
			return true
		}
	}
	return false
}

// HasAnyRoles 等价于 PermissionService.hasAnyRoles（分隔符 Constants.ROLE_DELIMITER）。
func HasAnyRoles(roleKeys []string, any string) bool {
	if strings.TrimSpace(any) == "" {
		return false
	}
	for _, r := range strings.Split(any, config.RoleDelimiter) {
		if HasRole(roleKeys, r) {
			return true
		}
	}
	return false
}

// Permi 权限校验中间件：不通过时返回 403 + "没有权限，请联系管理员授权"
func Permi(permission string) gin.HandlerFunc {
	return check(func(lu *entity.LoginUser) bool {
		return HasPermi(lu.Permissions, permission)
	}, permission)
}

// AnyPermi 任一权限校验中间件。
func AnyPermi(permissions string) gin.HandlerFunc {
	return check(func(lu *entity.LoginUser) bool {
		return HasAnyPermi(lu.Permissions, permissions)
	}, permissions)
}

// Role 角色校验中间件。
func Role(role string) gin.HandlerFunc {
	return check(func(lu *entity.LoginUser) bool {
		return HasRole(roleKeys(lu), role)
	}, "")
}

// AnyRoles 任一角色校验中间件。
func AnyRoles(roles string) gin.HandlerFunc {
	return check(func(lu *entity.LoginUser) bool {
		return HasAnyRoles(roleKeys(lu), roles)
	}, "")
}

func check(pass func(*entity.LoginUser) bool, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		lu, ok := LoginUser(c)
		if !ok {
			// 未经过 JWTAuth 或登录态缺失，按 401 处理
			unauthorized(c)
			return
		}
		if !pass(lu) {
			c.JSON(http.StatusOK, result.ErrorCode(config.StatusForbidden, config.MsgNoPermission))
			c.Abort()
			return
		}
		if permission != "" {
			// 等价于 PermissionContextHolder.setContext，供后续 @DataScope 等价逻辑取用
			c.Set(CtxPermission, permission)
		}
		c.Next()
	}
}

func roleKeys(lu *entity.LoginUser) []string {
	if lu == nil || lu.User == nil {
		return nil
	}
	return lu.User.RoleKeys()
}
