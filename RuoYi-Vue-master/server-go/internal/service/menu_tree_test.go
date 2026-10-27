package service

import (
	"testing"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// menus4Level 造一棵 4 层菜单树，形状照库里的真实数据：
// 系统管理(1) > 日志管理(108, M) > 操作日志(1080) > 详情按钮(10800)。
//
// 注意顺序是**打乱**的（子在父之前），逼实现必须真的递归而不是依赖输入顺序。
func menus4Level() []entity.SysMenu {
	return []entity.SysMenu{
		{MenuID: 10800, ParentID: 1080, MenuName: "操作日志详情", MenuType: "F"},
		{MenuID: 1080, ParentID: 108, MenuName: "操作日志", MenuType: "C"},
		{MenuID: 108, ParentID: 1, MenuName: "日志管理", MenuType: "M"},
		{MenuID: 100, ParentID: 1, MenuName: "用户管理", MenuType: "C"},
		{MenuID: 1, ParentID: 0, MenuName: "系统管理", MenuType: "M"},
		{MenuID: 2, ParentID: 0, MenuName: "系统监控", MenuType: "M"},
	}
}

// TestBuildMenuTreeDeep 回归用例：第三层及更深的 children 不能丢。
//
// 背景（实测 bug）：recursionFn 原来是
//
//	menu.Children = children
//	for i := range children { child := children[i]; recursionFn(list, &child) }
//
// Go 的 getChildList 返回**值拷贝**，对 &child 的修改不会回写到 children[i]，
// 于是第三层往下全部丢失。库里 sys_menu 最大层级是 4，
// getRouters 会丢掉三级路由、菜单树深层节点出不来。
func TestBuildMenuTreeDeep(t *testing.T) {
	tree := BuildMenuTree(menus4Level())

	require.Len(t, tree, 2, "两个根节点：系统管理、系统监控")
	assert.Equal(t, int64(1), tree[0].MenuID)
	assert.Equal(t, int64(2), tree[1].MenuID)

	sys := tree[0]
	require.Len(t, sys.Children, 2, "系统管理下有 用户管理 与 日志管理")

	var logs *entity.SysMenu
	for i := range sys.Children {
		if sys.Children[i].MenuID == 108 {
			logs = &sys.Children[i]
		}
	}
	require.NotNil(t, logs, "日志管理必须在系统管理下")

	// 第三层
	require.Len(t, logs.Children, 1, "第三层：操作日志")
	assert.Equal(t, int64(1080), logs.Children[0].MenuID)

	// 第四层 —— 修复前这里是 0（children 被值拷贝吃掉）
	require.Len(t, logs.Children[0].Children, 1,
		"第四层丢失即 recursionFn 的值拷贝 bug 复发了")
	assert.Equal(t, int64(10800), logs.Children[0].Children[0].MenuID)

	// 叶子不再有 children
	assert.Empty(t, logs.Children[0].Children[0].Children)
}

// TestBuildMenuTreeFilteredSubtree 过滤出的子树，其根要被识别为顶级节点。
//
// Java 的判定是 "parentId 不在本批 menuId 集合里"（不是 parentId == 0），
// 照抄 parentId == 0 会让按条件过滤后的子树整棵消失。
func TestBuildMenuTreeFilteredSubtree(t *testing.T) {
	sub := []entity.SysMenu{
		{MenuID: 1080, ParentID: 108, MenuName: "操作日志"},
		{MenuID: 108, ParentID: 1, MenuName: "日志管理"},
	}
	tree := BuildMenuTree(sub)
	require.Len(t, tree, 1, "只有 日志管理 是根（108 不在本批 menuId 里）")
	assert.Equal(t, int64(108), tree[0].MenuID)
	require.Len(t, tree[0].Children, 1)
	assert.Equal(t, int64(1080), tree[0].Children[0].MenuID)
}

func TestBuildMenuTreeSelect(t *testing.T) {
	ts := BuildMenuTreeSelect(menus4Level())
	require.Len(t, ts, 2)
	assert.Equal(t, int64(1), ts[0].ID)
	assert.Equal(t, "系统管理", ts[0].Label)
	assert.False(t, ts[0].Disabled)

	// children 的顺序跟随输入列表顺序（getChildList 顺序扫描），
	// 108 在输入里排在 100 之前，故这里是 Children[0]。
	logs := ts[0].Children[0]
	require.Equal(t, int64(108), logs.ID)
	require.Len(t, logs.Children, 1)
	require.Len(t, logs.Children[0].Children, 1, "TreeSelect 同样要保住第四层")
}
