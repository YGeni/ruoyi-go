package entity

import (
	"github.com/YGeni/ruoyi-go/server-go/internal/model"
)

// SysMenu 菜单权限表 sys_menu。
//
// 注意：is_frame / is_cache / menu_type / visible / status 在 Java 实体中
// 全部是 String（SysMenu.java:49-61），与 char(1) 列一致，Go 侧同样用 string，
// 禁止用 int 或 bool（否则 isFrame.equals("1") 这类判定会全部失效）。
type SysMenu struct {
	model.BaseModel
	MenuID    int64  `gorm:"column:menu_id;primaryKey;autoIncrement" json:"menuId"`
	MenuName  string `gorm:"column:menu_name" json:"menuName"`
	ParentID  int64  `gorm:"column:parent_id" json:"parentId"`
	OrderNum  int    `gorm:"column:order_num" json:"orderNum"`
	Path      string `gorm:"column:path" json:"path"`
	Component string `gorm:"column:component" json:"component"`
	Query     string `gorm:"column:query" json:"query"`
	RouteName string `gorm:"column:route_name" json:"routeName"` // 3.9 新增列
	IsFrame   string `gorm:"column:is_frame" json:"isFrame"`     // char(1)：'0'是外链 '1'否
	IsCache   string `gorm:"column:is_cache" json:"isCache"`     // char(1)：'0'缓存 '1'不缓存
	MenuType  string `gorm:"column:menu_type" json:"menuType"`   // M目录 C菜单 F按钮
	Visible   string `gorm:"column:visible" json:"visible"`      // char(1)：'0'显示 '1'隐藏
	Status    string `gorm:"column:status" json:"status"`        // char(1)：'0'正常 '1'停用
	Perms     string `gorm:"column:perms" json:"perms"`
	Icon      string `gorm:"column:icon" json:"icon"`

	// Children 仅用于内存构树，不对应数据库列。
	Children []SysMenu `gorm:"-" json:"children,omitempty"`
}

// TableName 表名。
func (SysMenu) TableName() string { return "sys_menu" }

// MetaVo 路由显示信息，等价于 ruoyi-system/.../domain/vo/MetaVo.java:15-30。
type MetaVo struct {
	Title   string `json:"title"`
	Icon    string `json:"icon"`
	NoCache bool   `json:"noCache"`
	Link    string `json:"link,omitempty"`
}

// RouterVo 路由配置，等价于 ruoyi-system/.../domain/vo/RouterVo.java:11-57。
//
// Java 侧类上有 @JsonInclude(JsonInclude.Include.NON_EMPTY)
// （RouterVo.java:11），因此字符串空值不输出；boolean 基本类型（hidden）恒输出。
// 字段顺序与取值必须满足前端 src/store/modules/permission.js:59-97 的解析方式。
type RouterVo struct {
	Name       string     `json:"name,omitempty"`
	Path       string     `json:"path,omitempty"`
	Hidden     bool       `json:"hidden"`
	Redirect   string     `json:"redirect,omitempty"`
	Component  string     `json:"component,omitempty"`
	Query      string     `json:"query,omitempty"`
	AlwaysShow *bool      `json:"alwaysShow,omitempty"`
	Meta       *MetaVo    `json:"meta,omitempty"`
	Children   []RouterVo `json:"children,omitempty"`
}
