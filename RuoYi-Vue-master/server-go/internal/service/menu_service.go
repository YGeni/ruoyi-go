package service

import (
	"strings"
	"unicode"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
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
	children := getChildList(list, menu.MenuID)
	menu.Children = children
	for i := range children {
		if len(getChildList(list, children[i].MenuID)) > 0 {
			child := children[i]
			recursionFn(list, &child)
		}
	}
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
