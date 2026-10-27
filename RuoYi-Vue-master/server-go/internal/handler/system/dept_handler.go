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

// DeptHandler 部门管理，对应 SysDeptController.java。
type DeptHandler struct{}

// List GET /system/dept/list
//
// 等价 SysDeptController.java:52-59：startPage() + selectDeptList + getDataTable。
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
	rows, total, err := service.SelectDeptList(c, lu, &q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysDept{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// ListExclude GET /system/dept/list/exclude/:deptId
//
// 等价 SysDeptController.java:65-73：排除该部门及其子部门后返回剩余部门。
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
	rows, _, err := service.SelectDeptList(c, lu, &q, &page.PageDomain{PageNum: 1, PageSize: 10000})
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
//
// 等价 SysDeptController.java:80-88。
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
// 等价 SysDeptController.java:110-124：checkDeptNameUnique -> insertDept。
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
//
// 等价 SysDeptController.java:130-149。
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
// 等价 SysDeptController.java:155-166：只改 order_num。
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
// 等价 SysDeptController.java:172-185：
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
