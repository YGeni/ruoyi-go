package system

import (
	"strconv"
	"strings"

	"github.com/YGeni/ruoyi-go/server-go/internal/middleware"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

// parseIDs 把 "1,2,3" 形式的路径/表单参数解析为 int64 切片，
// 等价于 Java 的 @PathVariable Long[] ids（Spring 按逗号切分）。
//
// 非法片段直接跳过：Java 侧会抛 MethodArgumentTypeMismatchException（400），
// Go 侧选择忽略，避免出现半截删除。空结果返回 nil。
func parseIDs(s string) []int64 {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]int64, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// loginUser 取当前登录用户，缺失时直接回 401。
//
// 所有需要 @DataScope 的接口都必须走这里：DataScope 依赖 LoginUser 里的
// User.Roles[].Permissions（见 service/datascope.go 的说明）。
func loginUser(c *gin.Context) (*entity.LoginUser, bool) {
	lu, ok := middleware.LoginUser(c)
	if !ok || lu == nil {
		result.Fail(c, result.ErrorCode(401, "认证失败，无法访问系统资源"))
		return nil, false
	}
	return lu, true
}

// mustLoginUser 取当前登录用户，调用前已确保存在（不回包）。
//
// 用于"已经调用过 loginUser 但需要再取一次"的场合（如各 update 接口的权限校验）。
func mustLoginUser(c *gin.Context) *entity.LoginUser {
	lu, _ := middleware.LoginUser(c)
	return lu
}

// usernameOf 取当前用户名，等价 BaseController.getUsername()。
//
// LoginUser 本身没有 UserName 字段（见 entity/sys_misc.go），
// 真实用户名在其内嵌的 User 上；取不到时回退空串（不阻断写入）。
func usernameOf(lu *entity.LoginUser) string {
	if lu == nil || lu.User == nil {
		return ""
	}
	return lu.User.UserName
}

// fail 统一输出业务失败体。
func fail(c *gin.Context, err error) {
	result.Fail(c, result.Error(err.Error()))
}
