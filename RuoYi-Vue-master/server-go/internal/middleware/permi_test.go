package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

// TestPermissionFunctions 表驱动覆盖 hasPermi / hasAnyPermi / hasRole / hasAnyRoles，
// 对应 ruoyi-framework/.../web/service/PermissionService.java:27-158。
func TestPermissionFunctions(t *testing.T) {
	cases := []struct {
		name     string
		check    func() bool
		wantPass bool
	}{
		// --- hasPermi（PermissionService.java:27-40 / 155-158）---
		{name: "普通权限命中", check: func() bool { return HasPermi([]string{"system:user:list", "system:role:list"}, "system:user:list") }, wantPass: true},
		{name: "权限串带空格需 trim", check: func() bool { return HasPermi([]string{"system:user:list"}, " system:user:list ") }, wantPass: true},
		{name: "*:*:* 直通", check: func() bool { return HasPermi([]string{"*:*:*"}, "system:user:list") }, wantPass: true},
		{name: "*:*:* 直通任意权限", check: func() bool { return HasPermi([]string{"*:*:*"}, "monitor:job:remove") }, wantPass: true},
		{name: "无权限", check: func() bool { return HasPermi([]string{"system:user:list"}, "system:user:remove") }, wantPass: false},
		{name: "权限集为空", check: func() bool { return HasPermi(nil, "system:user:list") }, wantPass: false},
		{name: "待校验权限为空", check: func() bool { return HasPermi([]string{"*:*:*"}, "") }, wantPass: false},
		// --- hasAnyPermi（PermissionService.java:59-80）---
		{name: "anyPermi 命中其一", check: func() bool { return HasAnyPermi([]string{"system:user:add"}, "system:user:edit,system:user:add") }, wantPass: true},
		{name: "anyPermi 全不命中", check: func() bool { return HasAnyPermi([]string{"system:user:add"}, "system:user:edit,system:user:remove") }, wantPass: false},
		{name: "anyPermi 遇 *:*:* 直通", check: func() bool { return HasAnyPermi([]string{"*:*:*"}, "a:b:c,d:e:f") }, wantPass: true},
		// --- hasRole（PermissionService.java:88-108）---
		{name: "角色命中", check: func() bool { return HasRole([]string{"common"}, "common") }, wantPass: true},
		{name: "admin 直通", check: func() bool { return HasRole([]string{"admin"}, "common") }, wantPass: true},
		{name: "角色不命中", check: func() bool { return HasRole([]string{"common"}, "admin2") }, wantPass: false},
		// --- hasAnyRoles（PermissionService.java:127-146）---
		{name: "anyRoles 命中其一", check: func() bool { return HasAnyRoles([]string{"common"}, "admin2,common") }, wantPass: true},
		{name: "anyRoles 全不命中", check: func() bool { return HasAnyRoles([]string{"common"}, "admin2,test") }, wantPass: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.check(); got != tc.wantPass {
				t.Fatalf("期望 %v，实际 %v", tc.wantPass, got)
			}
		})
	}
}

// TestPermiMiddleware 覆盖中间件要求的三种情形：
//  1. 普通权限命中 -> 放行
//  2. *:*:* 直通 -> 放行
//  3. 无权限 -> 403 且 msg 为 "没有权限，请联系管理员授权"
//     （对齐 GlobalExceptionHandler.java:35-41）
func TestPermiMiddleware(t *testing.T) {
	cases := []struct {
		name     string
		perms    []string
		perm     string
		wantCode int
		wantMsg  string
	}{
		{name: "普通权限命中放行", perms: []string{"system:user:list"}, perm: "system:user:list", wantCode: config.StatusSuccess},
		{name: "*:*:* 直通放行", perms: []string{"*:*:*"}, perm: "system:user:remove", wantCode: config.StatusSuccess},
		{name: "无权限返回403", perms: []string{"system:user:list"}, perm: "system:user:remove",
			wantCode: config.StatusForbidden, wantMsg: config.MsgNoPermission},
	}

	gin.SetMode(gin.TestMode)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lu := &entity.LoginUser{UserID: 2, Permissions: tc.perms, User: &entity.SysUser{UserID: 2}}
			r := gin.New()
			r.Use(func(c *gin.Context) { c.Set(CtxLoginUser, lu); c.Next() })
			r.GET("/test", Permi(tc.perm), func(c *gin.Context) {
				result.OK(c, result.Success())
			})

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("响应不是合法 JSON: %v，body=%s", err, w.Body.String())
			}
			gotCode, _ := body["code"].(float64)
			if int(gotCode) != tc.wantCode {
				t.Fatalf("code 期望 %d，实际 %v（body=%s）", tc.wantCode, gotCode, w.Body.String())
			}
			if tc.wantMsg != "" {
				gotMsg, _ := body["msg"].(string)
				if gotMsg != tc.wantMsg {
					t.Fatalf("msg 期望 %q，实际 %q", tc.wantMsg, gotMsg)
				}
			}
		})
	}
}

// TestPermiMiddlewareWithoutLogin 未携带登录态时应返回 401，而不是被放行。
func TestPermiMiddlewareWithoutLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", Permi("system:user:list"), func(c *gin.Context) { result.OK(c, result.Success()) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	gotCode, _ := body["code"].(float64)
	if int(gotCode) != config.StatusUnauthorized {
		t.Fatalf("code 期望 %d，实际 %v（body=%s）", config.StatusUnauthorized, gotCode, w.Body.String())
	}
}
