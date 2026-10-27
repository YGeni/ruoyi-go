package system

import (
	"strconv"
	"strings"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/service"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

// MenuHandler 菜单管理，对应 SysMenuController.java。
type MenuHandler struct{}

// List GET /system/menu/list
//
// 等价 SysMenuController.java:41-46。**不分页**（Java 的 list() 没有 startPage()），
// 返回体是 {code,msg,data:[...]}，前端 src/views/system/menu/index.vue:346
// 直接 handleTree(response.data, "menuId") 自己构树。
func (MenuHandler) List(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	var q query.MenuList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	rows, err := service.SelectMenuList(lu, &q)
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysMenu{}
	}
	result.OK(c, result.SuccessData(rows))
}

// GetInfo GET /system/menu/:menuId
//
// 等价 SysMenuController.java:52-56。
func (MenuHandler) GetInfo(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("menuId"), 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
	m, err := service.SelectMenuByID(id)
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(m))
}

// TreeSelect GET /system/menu/treeselect
//
// 等价 SysMenuController.java:61-66：selectMenuList + buildMenuTreeSelect，
// 返回 data 是 TreeSelect 数组（{id,label,children}）。
func (MenuHandler) TreeSelect(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	var q query.MenuList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	rows, err := service.SelectMenuList(lu, &q)
	if err != nil {
		fail(c, err)
		return
	}
	trees := service.BuildMenuTreeSelect(rows)
	if trees == nil {
		trees = []entity.TreeSelect{}
	}
	result.OK(c, result.SuccessData(trees))
}

// RoleMenuTreeSelect GET /system/menu/roleMenuTreeselect/:roleId
//
// 等价 SysMenuController.java:71-79：返回 {menus, checkedKeys}。
// 前端 src/views/system/role/index.vue:452-453 读 response.menus，
// 树用 node-key="id" + props {label:'label'}.
func (MenuHandler) RoleMenuTreeSelect(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	roleID, err := strconv.ParseInt(c.Param("roleId"), 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
	rows, err := service.SelectMenuList(lu, &query.MenuList{})
	if err != nil {
		fail(c, err)
		return
	}
	checked, err := service.SelectMenuIDsByRoleID(roleID)
	if err != nil {
		fail(c, err)
		return
	}
	if checked == nil {
		checked = []int64{}
	}
	result.OK(c, result.Success().
		Put("menus", service.BuildMenuTreeSelect(rows)).
		Put("checkedKeys", checked))
}

// Add POST /system/menu
//
// 等价 SysMenuController.java:87-103。
func (MenuHandler) Add(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	var m entity.SysMenu
	if err := c.ShouldBindJSON(&m); err != nil {
		fail(c, err)
		return
	}
	if err := service.InsertMenu(lu, &m); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// Edit PUT /system/menu
//
// 等价 SysMenuController.java:111-131。
func (MenuHandler) Edit(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	var m entity.SysMenu
	if err := c.ShouldBindJSON(&m); err != nil {
		fail(c, err)
		return
	}
	if err := service.UpdateMenu(lu, &m); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// UpdateSort PUT /system/menu/updateSort
//
// 等价 SysMenuController.java:139-145：body 是 Map<String,String>，
// menuIds 与 orderNums 都是**逗号分隔字符串**（不是数组）。
func (MenuHandler) UpdateSort(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var body struct {
		MenuIDs   string `json:"menuIds" form:"menuIds"`
		OrderNums string `json:"orderNums" form:"orderNums"`
	}
	if err := c.ShouldBind(&body); err != nil {
		fail(c, err)
		return
	}
	if err := service.UpdateMenuSort(
		strings.Split(body.MenuIDs, ","),
		strings.Split(body.OrderNums, ","),
	); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// Remove DELETE /system/menu/:menuId
//
// 等价 SysMenuController.java:153-164：有子菜单 / 已分配角色都返回 warn。
func (MenuHandler) Remove(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("menuId"), 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
	if err := service.DeleteMenu(id); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}
