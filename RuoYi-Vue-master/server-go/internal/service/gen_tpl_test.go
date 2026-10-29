package service

// 本文件守护的是 gentpl 的新行为：**生成 Go 代码，不再生成 Java**。
//
// 模板是字符串拼的，字段改名（比如 GoType 拼成 GoTypes）不会编译报错，
// 只会渲染出空值、生成一堆没有类型的 Go 代码。所以这里逐项断言生成结果。

import (
	"strings"
	"testing"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
)

// sampleGenTable 一张 sys_post 形状的生成表。
func sampleGenTable() *entity.GenTable {
	col := func(name, comment, colType, javaType, field string, pk, inc, ins, edit, list, query bool, qt string) entity.GenTableColumn {
		return entity.GenTableColumn{
			ColumnName:    name,
			ColumnComment: comment,
			ColumnType:    colType,
			JavaType:      javaType,
			JavaField:     field,
			IsPk:          b2s(pk),
			IsIncrement:   b2s(inc),
			IsInsert:      b2s(ins),
			IsEdit:        b2s(edit),
			IsList:        b2s(list),
			IsQuery:       b2s(query),
			QueryType:     qt,
			HTMLType:      HTMLInput,
		}
	}
	return &entity.GenTable{
		TableName:      "sys_post",
		TableComment:   "岗位信息表",
		ClassName:      "SysPost",
		TplCategory:    "crud",
		ModuleName:     "system",
		BusinessName:   "post",
		FunctionName:   "岗位",
		FunctionAuthor: "smoke",
		Columns: []entity.GenTableColumn{
			col("post_id", "岗位ID", "bigint", TypeLong, "postId", true, true, false, false, false, false, "EQ"),
			col("post_code", "岗位编码", "varchar(64)", TypeString, "postCode", false, false, true, true, true, true, "LIKE"),
			col("post_name", "岗位名称", "varchar(50)", TypeString, "postName", false, false, true, true, true, true, "LIKE"),
			col("post_sort", "显示顺序", "int", TypeInteger, "postSort", false, false, true, true, true, false, "EQ"),
			// status 走 EQ 分支（post_code/post_name 走 LIKE 分支），两条都要覆盖到
			col("status", "状态", "char(1)", TypeString, "status", false, false, true, true, true, true, "EQ"),
			// BaseModel 已带，实体里不应重复出现
			col("create_time", "创建时间", "datetime", TypeDate, "createTime", false, false, false, false, false, false, "EQ"),
		},
	}
}

// TestGenRendersGoNotJava 生成的必须是 Go 文件，且落在本项目的目录里。
func TestGenRendersGoNotJava(t *testing.T) {
	files, err := RenderAll(sampleGenTable())
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}
	want := []string{
		"internal/model/entity/sys_post.go",
		"internal/model/query/sys_post.go",
		"internal/repository/sys_post_repo.go",
		"internal/service/sys_post_service.go",
		"internal/handler/system/sys_post_handler.go",
		"internal/router/sys_post_routes.txt",
		"sql/sys_post.sql",
	}
	for _, w := range want {
		if _, ok := files[w]; !ok {
			keys := make([]string, 0, len(files))
			for k := range files {
				keys = append(keys, k)
			}
			t.Fatalf("缺少 %s，实际生成: %v", w, keys)
		}
	}
	for k, v := range files {
		// 这条是最关键的：不能再冒出 .java / .xml
		if strings.Contains(k, ".java") || strings.Contains(k, "mapper.xml") || strings.Contains(k, "main/java") {
			t.Errorf("生成了 Java 产物: %s", k)
		}
		if strings.Contains(v, "package com.ruoyi") || strings.Contains(v, "import java.") {
			t.Errorf("%s 里还有 Java 代码", k)
		}
	}
}

// TestGenEntityContent 实体：字段类型、gorm/json 标签、BaseModel 去重。
func TestGenEntityContent(t *testing.T) {
	files, err := RenderAll(sampleGenTable())
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}
	e := files["internal/model/entity/sys_post.go"]
	for _, want := range []string{
		"package entity",
		"type SysPost struct",
		"model.BaseModel",
		`gorm:"column:post_id;primaryKey;autoIncrement" json:"postId"`,
		`gorm:"column:post_code" json:"postCode"`,
		"PostSort int ",
		`func (SysPost) TableName() string { return "sys_post" }`,
	} {
		if !strings.Contains(e, want) {
			t.Errorf("entity 里缺 %q\n--- 实际 ---\n%s", want, e)
		}
	}
	// create_time 属于 BaseModel，不能重复生成
	if strings.Contains(e, `gorm:"column:create_time"`) {
		t.Errorf("create_time 是 BaseModel 字段，不该重复出现在实体里:\n%s", e)
	}
}

// TestGenRepositoryAndHandler 查询条件与 handler 的关键片段。
func TestGenRepositoryAndHandler(t *testing.T) {
	files, err := RenderAll(sampleGenTable())
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}
	repo := files["internal/repository/sys_post_repo.go"]
	for _, want := range []string{
		"func SelectSysPostList",
		"func SelectSysPostByID",
		"func InsertSysPost",
		"func UpdateSysPost",
		"func DeleteSysPostByIDs",
		// query_type=LIKE 的字符串列要模糊匹配
		`tx.Where("post_code LIKE ?", "%"+q.PostCode+"%")`,
		// EQ 的列用等号
		`tx.Where("status = ?", q.Status)`,
	} {
		if !strings.Contains(repo, want) {
			t.Errorf("repository 里缺 %q\n--- 实际 ---\n%s", want, repo)
		}
	}

	h := files["internal/handler/system/sys_post_handler.go"]
	for _, want := range []string{
		"package system",
		"type SysPostHandler struct{}",
		"func (SysPostHandler) List",
		"func (SysPostHandler) GetInfo",
		"func (SysPostHandler) Add",
		"func (SysPostHandler) Edit",
		"func (SysPostHandler) Remove",
		"service.SelectSysPostList(&q, page.FromRequest(c))",
		`strconv.ParseInt(c.Param("postId"), 10, 64)`,
		`parseIDs(c.Param("postIds"))`,
	} {
		if !strings.Contains(h, want) {
			t.Errorf("handler 里缺 %q\n--- 实际 ---\n%s", want, h)
		}
	}

	rt := files["internal/router/sys_post_routes.txt"]
	for _, want := range []string{
		"PermSystemPostList",
		`"/system/post/list"`,
		`"/system/post/:postId"`,
		`"/system/post/:postIds"`,
	} {
		if !strings.Contains(rt, want) {
			t.Errorf("routes.txt 里缺 %q", want)
		}
	}
}

// TestGenTypeMapping 类型映射：Java 类型 -> Go 类型，认不出不能瞎猜。
func TestGenTypeMapping(t *testing.T) {
	cases := map[string]string{
		TypeString:  "string",
		TypeLong:    "int64",
		TypeInteger: "int",
		TypeDouble:  "float64",
		TypeDate:    "*time.Time",
		TypeBoolean: "bool",
	}
	for in, want := range cases {
		if got := goType(in); got != want {
			t.Errorf("goType(%s) = %s，应为 %s", in, got, want)
		}
	}
	for in, want := range map[string]string{
		TypeString:  `""`,
		TypeDate:    "nil",
		TypeLong:    "0",
		TypeBoolean: "false",
	} {
		if got := goZero(in); got != want {
			t.Errorf("goZero(%s) = %s，应为 %s", in, got, want)
		}
	}
	if got := toSnake("SysPost"); got != "sys_post" {
		t.Errorf("toSnake(SysPost) = %s", got)
	}
	if got := classNameCap("SysPost"); got != "Post" {
		t.Errorf("classNameCap(SysPost) = %s", got)
	}
}
