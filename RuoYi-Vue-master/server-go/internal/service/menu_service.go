package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
)

// MENU_ROOT_ID 顶级菜单父 ID，等价于 SysMenuServiceImpl 的常量。
const MENU_ROOT_ID int64 = 0

// SelectMenuTreeByUserID 等价于 SysMenuServiceImpl.selectMenuTreeByUserId(L139-151)：
// 管理员取全量菜单，其余按角色取，然后构造成树。
func SelectMenuTreeByUserID(userID int64) ([]entity.SysMenu, error) {
	var (
		menus []entity.SysMenu
		err   error
	)
	if userID == 1 { // SecurityUtils.isAdmin
		menus, err = repository.SelectMenuTreeAll()
	} else {
		menus, err = repository.SelectMenuTreeByUserID(userID)
	}
	if err != nil {
		return nil, err
	}
	return getChildPerms(menus, MENU_ROOT_ID), nil
}

// getChildPerms 等价于 SysMenuServiceImpl.getChildPerms(L546-560) + recursionFn(L568-580)，
// 把平铺列表构造成树（children 只在内存中存在，不落库）。
func getChildPerms(list []entity.SysMenu, parentID int64) []entity.SysMenu {
	var result []entity.SysMenu
	for i := range list {
		if list[i].ParentID == parentID {
			menu := list[i]
			recursionFn(list, &menu)
			result = append(result, menu)
		}
	}
	return result
}

func recursionFn(list []entity.SysMenu, menu *entity.SysMenu) {
	// **必须先递归再赋值**。
	//
	// 原实现是 `menu.Children = children` 之后再 `child := children[i]; recursionFn(&child)`——
	// Go 的 getChildList 返回的是**值拷贝**，对 &child 的修改不会回写到 children[i]，
	// 于是第三层及更深的 children 全部丢失（Java 版本是对象引用所以没问题）。
	// 实测 sys_menu 最大层级是 4（系统管理 > 日志管理 > 操作日志），
	// 该 bug 会让 getRouters 丢掉三级路由、菜单树的深层节点也出不来。
	children := getChildList(list, menu.MenuID)
	for i := range children {
		if len(getChildList(list, children[i].MenuID)) > 0 {
			recursionFn(list, &children[i])
		}
	}
	menu.Children = children
}

func getChildList(list []entity.SysMenu, menuID int64) []entity.SysMenu {
	var out []entity.SysMenu
	for i := range list {
		if list[i].ParentID == menuID {
			out = append(out, list[i])
		}
	}
	return out
}

// BuildMenus 等价于 SysMenuServiceImpl.buildMenus(L172-222)。
//
// 返回值必须能被前端 src/store/modules/permission.js:59-97 解析：
// component 取 Layout / ParentView / InnerLink 或 views 相对路径。
func BuildMenus(menus []entity.SysMenu) []entity.RouterVo {
	routers := make([]entity.RouterVo, 0, len(menus))
	for i := range menus {
		menu := menus[i]
		router := entity.RouterVo{
			Hidden:    menu.Visible == "1",
			Name:      getRouteName(menu),
			Path:      getRouterPath(menu),
			Component: getComponent(menu),
			Query:     menu.Query,
			Meta:      newMeta(menu.MenuName, menu.Icon, menu.IsCache == "1", menu.Path),
		}
		if len(menu.Children) > 0 && menu.MenuType == config.TypeDir {
			alwaysShow := true
			router.AlwaysShow = &alwaysShow
			router.Redirect = "noRedirect"
			router.Children = BuildMenus(menu.Children)
		} else if isMenuFrame(menu) {
			router.Meta = nil
			router.Children = []entity.RouterVo{{
				Path:      menu.Path,
				Component: menu.Component,
				Name:      getRouteName2(menu.RouteName, menu.Path),
				Meta:      newMeta(menu.MenuName, menu.Icon, menu.IsCache == "1", menu.Path),
				Query:     menu.Query,
			}}
		} else if menu.ParentID == MENU_ROOT_ID && isInnerLink(menu) {
			router.Meta = &entity.MetaVo{Title: menu.MenuName, Icon: menu.Icon}
			router.Path = "/"
			routerPath := innerLinkReplaceEach(menu.Path)
			router.Children = []entity.RouterVo{{
				Path:      routerPath,
				Component: config.InnerLink,
				Name:      getRouteName2(menu.RouteName, routerPath),
				Meta:      newMeta(menu.MenuName, menu.Icon, false, menu.Path),
			}}
		}
		routers = append(routers, router)
	}
	return routers
}

// newMeta 等价于 MetaVo 的四个构造器：link 仅在 path 为 http(s) 时填充。
func newMeta(title, icon string, noCache bool, path string) *entity.MetaVo {
	meta := &entity.MetaVo{Title: title, Icon: icon, NoCache: noCache}
	if strings.HasPrefix(path, config.HTTP) || strings.HasPrefix(path, config.HTTPS) {
		meta.Link = path
	}
	return meta
}

// getRouteName 等价于 SysMenuServiceImpl.getRouteName(L430-438)。
func getRouteName(menu entity.SysMenu) string {
	if isMenuFrame(menu) {
		return ""
	}
	return getRouteName2(menu.RouteName, menu.Path)
}

// getRouteName2 等价于 SysMenuServiceImpl.getRouteName(L447-451)：首字母大写。
func getRouteName2(name, path string) string {
	routerName := name
	if strings.TrimSpace(routerName) == "" {
		routerName = path
	}
	if routerName == "" {
		return ""
	}
	r := []rune(routerName)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// getRouterPath 等价于 SysMenuServiceImpl.getRouterPath(L459-479)。
func getRouterPath(menu entity.SysMenu) string {
	routerPath := menu.Path
	if menu.ParentID != MENU_ROOT_ID && isInnerLink(menu) {
		routerPath = innerLinkReplaceEach(routerPath)
	}
	if MENU_ROOT_ID == menu.ParentID && menu.MenuType == config.TypeDir && menu.IsFrame == config.NoFrame {
		routerPath = "/" + menu.Path
	} else if isMenuFrame(menu) {
		routerPath = "/"
	}
	return routerPath
}

// getComponent 等价于 SysMenuServiceImpl.getComponent(L487-503)。
func getComponent(menu entity.SysMenu) string {
	component := config.Layout
	if menu.Component != "" && !isMenuFrame(menu) {
		component = menu.Component
	} else if menu.Component == "" && menu.ParentID != MENU_ROOT_ID && isInnerLink(menu) {
		component = config.InnerLink
	} else if menu.Component == "" && isParentView(menu) {
		component = config.ParentView
	}
	return component
}

// isMenuFrame 等价于 SysMenuServiceImpl.isMenuFrame(L511-515)。
func isMenuFrame(menu entity.SysMenu) bool {
	return menu.ParentID == MENU_ROOT_ID && menu.MenuType == config.TypeMenu && menu.IsFrame == config.NoFrame
}

// isParentView 等价于 SysMenuServiceImpl.isParentView(L523-525)。
func isParentView(menu entity.SysMenu) bool {
	return menu.ParentID != MENU_ROOT_ID && menu.MenuType == config.TypeDir
}

// isInnerLink 等价于 SysMenuServiceImpl.isInnerLink(L534-536)。
func isInnerLink(menu entity.SysMenu) bool {
	return menu.IsFrame == config.NoFrame && (strings.HasPrefix(menu.Path, config.HTTP) || strings.HasPrefix(menu.Path, config.HTTPS))
}

// innerLinkReplaceEach 等价于 SysMenuServiceImpl.innerLinkReplaceEach(L613-617)：
// 把 http:// https:// www. 去掉，. 与 : 替换成 /。
func innerLinkReplaceEach(path string) string {
	replacer := strings.NewReplacer(
		config.HTTP, "", config.HTTPS, "", config.WWW, "", ".", "/", ":", "/")
	return replacer.Replace(path)
}

// ---------- 菜单管理（SysMenuServiceImpl）----------

// SelectMenuList 等价 SysMenuServiceImpl.selectMenuList(L74-84)：
// 管理员查全部，其余按角色查。
func SelectMenuList(lu *entity.LoginUser, q *query.MenuList) ([]entity.SysMenu, error) {
	if lu == nil || lu.User == nil {
		return nil, errors.New("未登录")
	}
	if lu.User.UserID == 1 { // SecurityUtils.isAdmin
		return repository.SelectMenuList(q)
	}
	return repository.SelectMenuListByUserID(q, lu.User.UserID)
}

// SelectMenuByID 按主键查菜单，等价 selectMenuById(L272-276)。
func SelectMenuByID(menuID int64) (*entity.SysMenu, error) {
	return repository.SelectMenuByID(menuID)
}

// BuildMenuTree 等价 SysMenuServiceImpl.buildMenuTree(L231-248)。
//
// 判定"顶级节点"的方式是 **parentId 不在本批 menuId 集合里**（不是 parentId==0），
// 因此按条件过滤出一个子树时，该子树的根也会被正确识别为顶级节点。
// 若一个顶级节点都没找到（tempList 不含任何节点的 parentId），原样返回整批。
func BuildMenuTree(menus []entity.SysMenu) []entity.SysMenu {
	ids := make(map[int64]bool, len(menus))
	for i := range menus {
		ids[menus[i].MenuID] = true
	}
	out := make([]entity.SysMenu, 0, len(menus))
	for i := range menus {
		menu := menus[i]
		if !ids[menu.ParentID] {
			recursionFn(menus, &menu)
			out = append(out, menu)
		}
	}
	if len(out) == 0 {
		return menus
	}
	return out
}

// BuildMenuTreeSelect 等价 buildMenuTreeSelect(L259-262)：建树后转成 TreeSelect。
func BuildMenuTreeSelect(menus []entity.SysMenu) []entity.TreeSelect {
	trees := BuildMenuTree(menus)
	out := make([]entity.TreeSelect, 0, len(trees))
	for i := range trees {
		out = append(out, toTreeSelect(trees[i]))
	}
	return out
}

func toTreeSelect(m entity.SysMenu) entity.TreeSelect {
	ts := entity.TreeSelect{ID: m.MenuID, Label: m.MenuName}
	for _, c := range m.Children {
		ts.Children = append(ts.Children, toTreeSelect(c))
	}
	return ts
}

// SelectMenuIDsByRoleID 等价 selectMenuListByRoleId(L160-165)：
// 先取角色的 menuCheckStrictly，再决定要不要排除"父菜单也在结果集里"的子菜单。
func SelectMenuIDsByRoleID(roleID int64) ([]int64, error) {
	role, err := repository.SelectRoleByID(roleID)
	if err != nil {
		return nil, err
	}
	strict := role != nil && role.MenuCheckStrictly
	return repository.SelectMenuIDsByRoleID(roleID, strict)
}

// isHTTP 等价 StringUtils.ishttp：以 http:// 或 https:// 开头。
func isHTTP(s string) bool {
	return strings.HasPrefix(s, config.HTTP) || strings.HasPrefix(s, config.HTTPS)
}

// checkMenuNameUnique 等价 checkMenuNameUnique(L383-393)：true 表示可用。
func checkMenuNameUnique(m *entity.SysMenu) (bool, error) {
	info, err := repository.CheckMenuNameUnique(m.MenuName, m.ParentID)
	if err != nil {
		return false, err
	}
	if info != nil && info.MenuID != m.MenuID {
		return false, nil
	}
	return true, nil
}

// checkRouteConfigUnique 等价 checkRouteConfigUnique(L397-428)：true 表示可用。
//
// 三条冲突规则逐条对齐：
//  1. 同级下 path 相同（parentId 相同且 path 相同，忽略大小写）
//  2. 根目录（parentId == 0）下 path 必须全局唯一
//  3. routeName 必须全局唯一（routeName 为空时取 path）
func checkRouteConfigUnique(m *entity.SysMenu) (bool, error) {
	routeName := m.RouteName
	if routeName == "" {
		routeName = m.Path
	}
	list, err := repository.SelectMenusByPathOrRouteName(m.Path, routeName)
	if err != nil {
		return false, err
	}
	for i := range list {
		it := list[i]
		if it.MenuID == m.MenuID {
			continue
		}
		dbRouteName := it.RouteName
		if dbRouteName == "" {
			dbRouteName = it.Path
		}
		switch {
		case strings.EqualFold(m.Path, it.Path) && m.ParentID == it.ParentID:
			return false, nil
		case strings.EqualFold(m.Path, it.Path) && m.ParentID == MENU_ROOT_ID:
			return false, nil
		case strings.EqualFold(routeName, dbRouteName):
			return false, nil
		}
	}
	return true, nil
}

// menuNameUniqueErr 菜单名重复的文案（SysMenuController.java:91/115）。
func menuNameUniqueErr(verb, name string) error {
	return fmt.Errorf("%s菜单'%s'失败，菜单名称已存在", verb, name)
}

// InsertMenu 新增菜单，等价 SysMenuController.add(L87-103) 的三条校验。
func InsertMenu(lu *entity.LoginUser, m *entity.SysMenu) error {
	m.CreateBy = usernameOf(lu)
	ok, err := checkMenuNameUnique(m)
	if err != nil {
		return err
	}
	if !ok {
		return menuNameUniqueErr("新增", m.MenuName)
	}
	if m.IsFrame == config.YesFrame && !isHTTP(m.Path) {
		return fmt.Errorf("新增菜单'%s'失败，地址必须以http(s)://开头", m.MenuName)
	}
	ok, err = checkRouteConfigUnique(m)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("新增菜单'%s'失败，路由名称或地址已存在", m.MenuName)
	}
	return repository.InsertMenu(m)
}

// UpdateMenu 修改菜单，等价 SysMenuController.edit(L111-131) 的四条校验。
func UpdateMenu(lu *entity.LoginUser, m *entity.SysMenu) error {
	m.UpdateBy = usernameOf(lu)
	ok, err := checkMenuNameUnique(m)
	if err != nil {
		return err
	}
	if !ok {
		return menuNameUniqueErr("修改", m.MenuName)
	}
	if m.IsFrame == config.YesFrame && !isHTTP(m.Path) {
		return fmt.Errorf("修改菜单'%s'失败，地址必须以http(s)://开头", m.MenuName)
	}
	if m.MenuID == m.ParentID {
		return fmt.Errorf("修改菜单'%s'失败，上级菜单不能选择自己", m.MenuName)
	}
	ok, err = checkRouteConfigUnique(m)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("修改菜单'%s'失败，路由名称或地址已存在", m.MenuName)
	}
	return repository.UpdateMenu(m)
}

// UpdateMenuSort 保存菜单排序，等价 updateMenuSort(L339-352)。
//
// Java 用 String[] split(",") 后逐个更新，任一转换失败抛 "保存排序异常，请联系管理员"。
func UpdateMenuSort(menuIDs, orderNums []string) error {
	if len(menuIDs) != len(orderNums) {
		return errors.New("保存排序异常，请联系管理员")
	}
	for i := range menuIDs {
		id, err := strconv.ParseInt(strings.TrimSpace(menuIDs[i]), 10, 64)
		if err != nil {
			return errors.New("保存排序异常，请联系管理员")
		}
		num, err := strconv.Atoi(strings.TrimSpace(orderNums[i]))
		if err != nil {
			return errors.New("保存排序异常，请联系管理员")
		}
		if err := repository.UpdateMenuSort(id, num); err != nil {
			return err
		}
	}
	return nil
}

// DeleteMenu 删除菜单，等价 SysMenuController.remove(L153-164) 的两条前置校验。
func DeleteMenu(menuID int64) error {
	has, err := repository.HasChildByMenuID(menuID)
	if err != nil {
		return err
	}
	if has {
		return errors.New("存在子菜单,不允许删除")
	}
	used, err := repository.CheckMenuExistRole(menuID)
	if err != nil {
		return err
	}
	if used {
		return errors.New("菜单已分配,不允许删除")
	}
	return repository.DeleteMenuByID(menuID)
}
