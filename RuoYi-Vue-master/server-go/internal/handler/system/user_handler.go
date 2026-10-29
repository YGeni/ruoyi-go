// Package system 对应 ruoyi-admin/src/main/java/com/ruoyi/web/controller/system。
package system

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/YGeni/ruoyi-go/server-go/internal/middleware"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/service"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户管理，对应。
type UserHandler struct{}

// List GET /system/user/list
//
//	startPage(); List<SysUser> list = userService.selectUserList(user); return getDataTable(list);
//
// 返回体必须是 TableDataInfo{total, rows, code:200, msg:"查询成功"}。
func (UserHandler) List(c *gin.Context) {
	lu, ok := middleware.LoginUser(c)
	if !ok {
		result.Fail(c, result.ErrorCode(401, "认证失败，无法访问系统资源"))
		return
	}
	var q query.UserList
	if err := c.ShouldBindQuery(&q); err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}
	q.FillDateRange(c)
	rows, total, err := service.SelectUserList(c, lu, &q, page.FromRequest(c))
	if err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}
	if rows == nil {
		rows = []entity.SysUser{}
	}
	// TableDataInfo{total, rows, code:200, msg:"查询成功"}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// GetInfo GET /system/user/ 与 GET /system/user/:userId
//
//	@GetMapping(value = { "/", "/{userId}" })
//	public AjaxResult getInfo(@PathVariable(value = "userId", required = false) Long userId)
//
// userId 为空时返回空对象（Java 侧同样返回 new SysUser()）。
func (UserHandler) GetInfo(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	s := strings.TrimSpace(c.Param("userId"))
	if s == "" {
		result.OK(c, result.SuccessData(entity.SysUser{}))
		return
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
	if err := service.CheckUserDataScope(mustLoginUser(c), id); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	u, err := service.SelectUserByID(id)
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(u))
}

// Add POST /system/user
//
// 唯一性校验 -> 密码加密 -> insertUser（含角色关联）。
func (UserHandler) Add(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var u entity.SysUser
	// Password 的 json tag 是 "-"（绝不从详情接口泄漏散列），
	// 因此新增/重置密码必须用独立字段接收，不能靠 SysUser 直接绑定。
	var pwd struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&u); err != nil {
		fail(c, err)
		return
	}
	_ = c.ShouldBindJSON(&pwd)
	u.Password = pwd.Password
	if err := service.InsertUser(&u, u.RoleIds); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// Edit PUT /system/user
//
// 等价：checkUserAllowed + checkUserDataScope + updateUser。
func (UserHandler) Edit(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var u entity.SysUser
	if err := c.ShouldBindJSON(&u); err != nil {
		fail(c, err)
		return
	}
	if err := service.CheckUserAllowed(u.UserID); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	if err := service.CheckUserDataScope(mustLoginUser(c), u.UserID); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	if err := service.UpdateUser(&u, u.RoleIds); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// Remove DELETE /system/user/:userIds
func (UserHandler) Remove(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	ids := parseIDs(c.Param("userIds"))
	if len(ids) == 0 {
		result.Fail(c, result.Error("未选择要删除的用户"))
		return
	}
	if err := service.DeleteUserByIDs(lu, ids); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// ResetPwd PUT /system/user/resetPwd
func (UserHandler) ResetPwd(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	// resetPwd 只认 userId + password 两个字段（Java 的 SysUserController.resetPwd
	// 接收整个 SysUser，但 password 是唯一生效的字段）。
	var body struct {
		UserID   int64  `json:"userId"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, err)
		return
	}
	if err := service.CheckUserAllowed(body.UserID); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	if err := service.CheckUserDataScope(mustLoginUser(c), body.UserID); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	if err := service.ResetUserPwd(body.UserID, body.Password); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// ChangeStatus PUT /system/user/changeStatus
func (UserHandler) ChangeStatus(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var u entity.SysUser
	if err := c.ShouldBindJSON(&u); err != nil {
		fail(c, err)
		return
	}
	if err := service.CheckUserAllowed(u.UserID); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	if err := service.CheckUserDataScope(mustLoginUser(c), u.UserID); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	if err := service.UpdateUserStatus(u.UserID, u.Status); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// AuthRole GET /system/user/authRole/:userId
//
// 等价：返回用户详情 + 全部角色 + 已分配角色 ID。
//
// 返回体：AjaxResult{user, roles, checkedKeys}（Java 用 AjaxResult.put 三次）。
func (UserHandler) AuthRole(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
	u, err := service.SelectUserByID(id)
	if err != nil {
		fail(c, err)
		return
	}
	roles, err := service.SelectRoleAll()
	if err != nil {
		fail(c, err)
		return
	}
	checked, err := service.SelectUserRoleIDs(id)
	if err != nil {
		fail(c, err)
		return
	}
	if checked == nil {
		checked = []int64{}
	}
	result.OK(c, result.Success().
		Put("user", u).
		Put("roles", roles).
		Put("checkedKeys", checked))
}

// InsertAuthRole PUT /system/user/authRole
//
// 等价：insertAuthRole(userId, roleIds)。
func (UserHandler) InsertAuthRole(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var body struct {
		UserID  int64   `json:"userId" form:"userId"`
		RoleIDs []int64 `json:"roleIds" form:"roleIds"`
	}
	if err := c.ShouldBind(&body); err != nil {
		fail(c, err)
		return
	}
	if err := service.InsertUserRoles(body.UserID, body.RoleIDs); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// DeptTree GET /system/user/deptTree
//
// 等价：返回部门树（前端自行组装成 el-tree 结构）。
// 权限串是 system:user:list（见 perm_table.go 的登记）。
func (UserHandler) DeptTree(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	rows, _, err := service.SelectDeptList(c, lu, &query.DeptList{},
		&page.PageDomain{PageNum: 1, PageSize: 10000})
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysDept{}
	}
	result.OK(c, result.SuccessData(rows))
}
