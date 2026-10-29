package system

import (
	"strconv"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/service"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

// DeptHandler 部门管理，对应。
type DeptHandler struct{}

// List GET /system/dept/list
//
// 等价：**没有 startPage，直接 success(depts)**。
//
// 返回体必须是 {code, msg, data:[...]}，不能是 getDataTable 的 {rows, total}。
// 前端 src/views/system/dept/index.vue:193 写的是
//
//	deptList.value = proxy.handleTree(response.data, "deptId")
//
// 之前这里错写成 DataTable，response.data 是 undefined，
// 前端 handleTree 里的 for...of 直接抛
// "undefined is not an object (evaluating 'd of data')"（ruoyi.js:167）。
//
// 该接口是 **DataScope 五个落点之一**（@DataScope(deptAlias="d")，无 userAlias）。
func (DeptHandler) List(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	var q query.DeptList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	rows, _, err := service.SelectDeptList(c, lu, &q, page.NoLimit())
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysDept{}
	}
	result.OK(c, result.SuccessData(rows))
}

// ListExclude GET /system/dept/list/exclude/:deptId
//
// 等价：排除该部门及其子部门后返回剩余部门。
//
// DIFF(review): Java 在 service 层循环 remove 掉 deptId 与 ancestors 命中它的行；
// Go 侧同样在内存里过滤，语义等价（部门数量有限）。
func (DeptHandler) ListExclude(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	excludeID, err := strconv.ParseInt(c.Param("deptId"), 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
	var q query.DeptList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	rows, _, err := service.SelectDeptList(c, lu, &q, page.NoLimit())
	if err != nil {
		fail(c, err)
		return
	}
	out := make([]entity.SysDept, 0, len(rows))
	for i := range rows {
		if rows[i].DeptID == excludeID || deptContains(rows[i].Ancestors, excludeID) {
			continue
		}
		out = append(out, rows[i])
	}
	result.OK(c, result.SuccessData(out))
}

// deptContains 判断 ancestors（逗号分隔）里是否含目标部门 ID，
// 等价 SysDeptServiceImpl 里 StringUtils.containsAny(dept.getAncestors(), ...) 的过滤。
func deptContains(ancestors string, deptID int64) bool {
	if ancestors == "" || deptID == 0 {
		return false
	}
	target := strconv.FormatInt(deptID, 10)
	for _, p := range splitComma(ancestors) {
		if p == target {
			return true
		}
	}
	return false
}

func splitComma(s string) []string {
	out := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// GetInfo GET /system/dept/:deptId
func (DeptHandler) GetInfo(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("deptId"), 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
	if err := service.CheckDeptDataScope(c, mustLoginUser(c), id); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	d, err := service.SelectDeptByID(id)
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(d))
}

// Add POST /system/dept
//
// 等价：checkDeptNameUnique -> insertDept。
func (DeptHandler) Add(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var d entity.SysDept
	if err := c.ShouldBindJSON(&d); err != nil {
		fail(c, err)
		return
	}
	ok, err := service.CheckDeptNameUnique(&d)
	if err != nil {
		fail(c, err)
		return
	}
	if !ok {
		result.Fail(c, result.Warn("新增部门'"+d.DeptName+"'失败，部门名称已存在"))
		return
	}
	if err := service.InsertDept(&d); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// Edit PUT /system/dept
func (DeptHandler) Edit(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var d entity.SysDept
	if err := c.ShouldBindJSON(&d); err != nil {
		fail(c, err)
		return
	}
	if err := service.CheckDeptDataScope(c, mustLoginUser(c), d.DeptID); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	ok, err := service.CheckDeptNameUnique(&d)
	if err != nil {
		fail(c, err)
		return
	}
	if !ok {
		result.Fail(c, result.Warn("修改部门'"+d.DeptName+"'失败，部门名称已存在"))
		return
	}
	if err := service.UpdateDept(&d); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// UpdateSort PUT /system/dept/updateSort
//
// 等价：只改 order_num。
func (DeptHandler) UpdateSort(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var d entity.SysDept
	if err := c.ShouldBindJSON(&d); err != nil {
		fail(c, err)
		return
	}
	if err := service.UpdateDeptSort(d.DeptID, d.OrderNum); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// Remove DELETE /system/dept/:deptId
//
// 存在下级部门 -> warn；部门存在用户 -> warn；否则删除。
func (DeptHandler) Remove(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("deptId"), 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
	if err := service.DeleteDeptByID(id); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}
