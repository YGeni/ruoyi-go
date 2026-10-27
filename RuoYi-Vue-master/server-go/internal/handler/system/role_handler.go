package system

import (
	"net/http"
	"strconv"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/service"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

// RoleHandler 角色管理，对应 SysRoleController.java。
type RoleHandler struct{}

// List GET /system/role/list
//
// 等价 SysRoleController.java:60-67：startPage() + selectRoleList + getDataTable。
// 该接口是 **DataScope 五个落点之一**（@DataScope(deptAlias="d")，无 userAlias）。
func (RoleHandler) List(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	var q query.RoleList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	q.FillDateRange(c)
	rows, total, err := service.SelectRoleList(c, lu, &q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysRole{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// OptionSelect GET /system/role/optionselect
//
// 等价 SysRoleController.java:178-183：返回全部正常角色，不带分页。
func (RoleHandler) OptionSelect(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	rows, err := service.SelectRoleAll()
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysRole{}
	}
	result.OK(c, result.SuccessData(rows))
}

// GetInfo GET /system/role/:roleId
//
// 等价 SysRoleController.java:98-105：success(roleService.selectRoleById(roleId))。
func (RoleHandler) GetInfo(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("roleId"), 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
	// 数据权限校验（SysRoleServiceImpl.checkRoleDataScope）
	if err := service.CheckRoleDataScope(mustLoginUser(c), id); err != nil {
		fail(c, err)
		return
	}
	r, err := service.SelectRoleByID(id)
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(r))
}

// roleByID 取路径参数指向的角色，并做 admin 保护校验。
func roleByID(c *gin.Context) (*entity.SysRole, bool) {
	id, err := strconv.ParseInt(c.Param("roleId"), 10, 64)
	if err != nil {
		fail(c, err)
		return nil, false
	}
	r, err := service.SelectRoleByID(id)
	if err != nil {
		fail(c, err)
		return nil, false
	}
	if r == nil {
		result.Fail(c, result.Error("角色不存在"))
		return nil, false
	}
	return r, true
}

// Add POST /system/role
//
// 等价 SysRoleController.java:110-118：
// 先 checkRoleAllowed/checkRoleNameUnique/checkRoleKeyUnique，再 insertRole + insertRoleMenu。
func (RoleHandler) Add(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var r entity.SysRole
	if err := c.ShouldBindJSON(&r); err != nil {
		fail(c, err)
		return
	}
	if err := service.CheckRoleAllowed(&r); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	menuIDs := r.MenuIds
	if err := service.InsertRole(&r, menuIDs); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// Edit PUT /system/role
//
// 等价 SysRoleController.java:125-136：checkRoleAllowed + updateRole + insertRoleMenu。
func (RoleHandler) Edit(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var r entity.SysRole
	if err := c.ShouldBindJSON(&r); err != nil {
		fail(c, err)
		return
	}
	exist, ok := roleForUpdate(c, r.RoleID)
	if !ok {
		return
	}
	if err := service.CheckRoleAllowed(exist); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	if err := service.CheckRoleAllowed(&r); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	menuIDs := r.MenuIds
	if err := service.UpdateRole(&r, menuIDs, r.MenuCheckStrictly, r.DeptCheckStrictly); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	if menuIDs != nil {
		if err := service.InsertRoleMenu(r.RoleID, menuIDs); err != nil {
			fail(c, err)
			return
		}
	}
	result.OK(c, result.Success())
}

// roleForUpdate 修改类接口共用的前置校验：角色存在 + 数据权限范围内。
func roleForUpdate(c *gin.Context, roleID int64) (*entity.SysRole, bool) {
	r, err := service.SelectRoleByID(roleID)
	if err != nil {
		fail(c, err)
		return nil, false
	}
	if r == nil {
		result.Fail(c, result.Error("角色不存在"))
		return nil, false
	}
	if err := service.CheckRoleDataScope(mustLoginUser(c), roleID); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return nil, false
	}
	return r, true
}

// DataScope PUT /system/role/dataScope
//
// 等价 SysRoleController.java:142-151：authDataScope（写 data_scope + sys_role_dept）。
func (RoleHandler) DataScope(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var body struct {
		RoleID    int64   `json:"roleId"`
		DataScope string  `json:"dataScope"`
		DeptIDs   []int64 `json:"deptIds"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, err)
		return
	}
	exist, err := service.SelectRoleByID(body.RoleID)
	if err != nil {
		fail(c, err)
		return
	}
	if err := service.CheckRoleAllowed(exist); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	if err := service.UpdateRoleDataScope(body.RoleID, body.DataScope, body.DeptIDs); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// ChangeStatus PUT /system/role/changeStatus
//
// 等价 SysRoleController.java:154-163。
func (RoleHandler) ChangeStatus(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var r entity.SysRole
	if err := c.ShouldBindJSON(&r); err != nil {
		fail(c, err)
		return
	}
	exist, ok := roleForUpdate(c, r.RoleID)
	if !ok {
		return
	}
	if err := service.CheckRoleAllowed(exist); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	if err := service.UpdateRoleStatus(r.RoleID, r.Status); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// Remove DELETE /system/role/:roleIds
//
// 等价 SysRoleController.java:168-176：先 checkRoleAllowed 再 deleteRoleByIds。
func (RoleHandler) Remove(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	ids := parseIDs(c.Param("roleIds"))
	if len(ids) == 0 {
		result.Fail(c, result.Error("未选择要删除的角色"))
		return
	}
	for _, id := range ids {
		r, err := service.SelectRoleByID(id)
		if err != nil {
			fail(c, err)
			return
		}
		if err := service.CheckRoleAllowed(r); err != nil {
			result.Fail(c, result.Warn(err.Error()))
			return
		}
	}
	if err := service.DeleteRoleByIDs(ids); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// DeptTree GET /system/role/deptTree/:roleId
//
// 等价 SysRoleController.java:246-253：返回该角色已授权的部门 ID 列表。
func (RoleHandler) DeptTree(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("roleId"), 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
	ids, err := service.SelectRoleDeptTree(id)
	if err != nil {
		fail(c, err)
		return
	}
	if ids == nil {
		ids = []int64{}
	}
	// Java 的 SysRoleController.java:250-251 会 put 两个键：checkedKeys 与 depts。
	// 前端 src/views/system/role/index.vue:458 读的是 response.depts，
	// 只回 checkedKeys 会让"数据权限" Tab 的部门树一片空白。
	lu := mustLoginUser(c)
	depts, _, err := service.SelectDeptList(c, lu, &query.DeptList{},
		&page.PageDomain{PageNum: 1, PageSize: 10000})
	if err != nil {
		fail(c, err)
		return
	}
	if depts == nil {
		depts = []entity.SysDept{}
	}
	// 注意 **必须挂在顶层，不能塞进 data**：
	// Java 的 AjaxResult.success() 只含 code/msg，两个 put 都是顶层键，
	// 前端 src/views/system/role/index.vue:458 读的是 response.depts。
	result.OK(c, result.Success().Put("checkedKeys", ids).Put("depts", depts))
}

// AllocatedList GET /system/role/authUser/allocatedList
//
// 等价 SysRoleController.java:188-198。该接口是 **DataScope 五个落点之一**
// （@DataScope(deptAlias="d", userAlias="u")，与 selectUserList 同别名组合）。
func (RoleHandler) AllocatedList(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	roleID, err := strconv.ParseInt(c.Param("roleId"), 10, 64)
	if err != nil {
		roleID, err = strconv.ParseInt(c.Query("roleId"), 10, 64)
		if err != nil {
			fail(c, err)
			return
		}
	}
	var q query.AuthUserList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	rows, total, err := service.SelectAllocatedList(c, lu, roleID, &q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysUser{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// UnallocatedList GET /system/role/authUser/unallocatedList
//
// 等价 SysRoleController.java:200-211。同样是 **DataScope 五个落点之一**。
func (RoleHandler) UnallocatedList(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	roleID, err := strconv.ParseInt(c.Param("roleId"), 10, 64)
	if err != nil {
		roleID, err = strconv.ParseInt(c.Query("roleId"), 10, 64)
		if err != nil {
			fail(c, err)
			return
		}
	}
	var q query.AuthUserList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	rows, total, err := service.SelectUnallocatedList(c, lu, roleID, &q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysUser{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// AuthUserCancel PUT /system/role/authUser/cancel
//
// 等价 SysRoleController.java:213-222：deleteAuthUser(userId, roleId)。
func (RoleHandler) AuthUserCancel(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var body struct {
		UserID int64 `json:"userId" form:"userId"`
		RoleID int64 `json:"roleId" form:"roleId"`
	}
	if err := c.ShouldBind(&body); err != nil {
		fail(c, err)
		return
	}
	if err := service.CancelAuthUser(body.UserID, body.RoleID); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// AuthUserCancelAll PUT /system/role/authUser/cancelAll
//
// 等价 SysRoleController.java:224-233。前端以 ?roleId=1&userIds=1,2,3 形式提交。
func (RoleHandler) AuthUserCancelAll(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	roleID, _ := strconv.ParseInt(c.Query("roleId"), 10, 64)
	userIDs := parseIDs(c.Query("userIds"))
	if err := service.CancelAllAuthUser(roleID, userIDs); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// AuthUserSelectAll PUT /system/role/authUser/selectAll
//
// 等价 SysRoleController.java:235-244。前端以 ?roleId=1&userIds=1,2,3 形式提交。
func (RoleHandler) AuthUserSelectAll(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	roleID, _ := strconv.ParseInt(c.Query("roleId"), 10, 64)
	userIDs := parseIDs(c.Query("userIds"))
	if err := service.InsertAuthUsers(roleID, userIDs); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}
