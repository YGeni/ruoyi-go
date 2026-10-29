// Package gentpl 代码生成的模板。
//
// DIFF(review) —— 与原始 RuoYi 的**根本差别**：Java 侧生成的是 Spring Boot
// 那套（domain / mapper.java / service / serviceImpl / controller / mapper.xml），
// 而这里生成的是**本项目的 Go 代码**，落点就是本项目的目录结构：
//
//	internal/model/entity/*.go        实体（gorm 标签 + json 标签）
//	internal/model/query/*.go         列表查询条件
//	internal/repository/*_repo.go     GORM 数据访问
//	internal/service/*_service.go     业务层
//	internal/handler/system/*.go      Gin handler
//	internal/router/*_routes.txt      要往 perm_table.go / router.go 里加的条目
//	sql/*.sql                         菜单 SQL（与语言无关，保持不变）
//
// 生成的 Go 代码是**照着本项目现有模块（如 SysPost）写的**：用 db.DB、
// page.Paginate、result.DataTable、loginUser/fail 这些既有设施，不是另起一套。
//
// 两点范围取舍：
//
//  1. **只支持 tpl_category = crud**。Velocity 模板里的 sub（子表）与 tree
//     （树表）分支不移植，这两类按 crud 渲染并在 options 里注明。
//  2. 路由与权限**不生成代码文件**，只给 routes.txt：因为本项目是
//     perm_table.go 统一登记 + middleware 校验，直接塞一个 .go 进 internal/router
//     要么编译不过（权限常量要先定义）、要么绕过对账测试。给人去加，反而安全。
package gentpl

import (
	"strings"
	"text/template"
)

// Files 模板文件名 -> 渲染出的文件路径（相对生成根目录）。
//
// 路径按本项目的目录结构落，不再是 Java 的 main/java/... 。
var Files = []struct {
	Tpl  string // 模板名
	Path string // 输出路径模板（用同样的字段渲染）
}{
	{"entity.go", "internal/model/entity/{{.SnakeName}}.go"},
	{"query.go", "internal/model/query/{{.SnakeName}}.go"},
	{"repository.go", "internal/repository/{{.SnakeName}}_repo.go"},
	{"service.go", "internal/service/{{.SnakeName}}_service.go"},
	{"handler.go", "internal/handler/system/{{.SnakeName}}_handler.go"},
	{"routes.txt", "internal/router/{{.SnakeName}}_routes.txt"},
	{"sql.sql", "sql/{{.TableName}}.sql"},
}

var raw = map[string]string{
	"entity.go": `package entity

{{if .HasTime}}import (
	"time"

	"{{.GoModule}}/internal/model"
)
{{else}}import "{{.GoModule}}/internal/model"
{{end}}
// {{.ClassName}} {{.TableComment}}（{{.TableName}}）。
//
// 由代码生成功能生成于 {{.Datetime}}，作者 {{.FunctionAuthor}}。
// 生成后请核对：字段类型、主键是否自增、以及 gorm 标签里该不该加索引。
type {{.ClassName}} struct {
	model.BaseModel
{{range .Columns}}{{if not .IsSuperColumn}}	{{.GoField}} {{.GoType}} {{tick}}gorm:"column:{{.ColumnName}}{{if .IsPk}};primaryKey{{if .IsIncrementBool}};autoIncrement{{end}}{{end}}" json:"{{.JavaField}}"{{tick}} // {{.ColumnComment}}
{{end}}{{end}}}

// TableName 表名。
func ({{.ClassName}}) TableName() string { return "{{.TableName}}" }
`,

	"query.go": `package query

// {{.ClassName}}List GET {{.PathPrefix}}/list 的查询条件。
//
// 由代码生成功能生成于 {{.Datetime}}。
// 只有 gen_table_column 里 is_query='1' 的列会出现在这里。
type {{.ClassName}}List struct {
{{range .QueryColumns}}	{{.GoField}} {{.GoType}} {{tick}}form:"{{.JavaField}}" json:"{{.JavaField}}"{{tick}} // {{.ColumnComment}}
{{end}}}
`,

	"repository.go": `package repository

import (
	"errors"

	"{{.GoModule}}/internal/model/entity"
	"{{.GoModule}}/internal/model/query"
	"{{.GoModule}}/pkg/db"
	"{{.GoModule}}/pkg/page"

	"gorm.io/gorm"
)

// {{.ClassNameLower}}ListQuery 拼列表查询条件（is_query='1' 的列）。
//
// 字符串列且 query_type=LIKE 的用模糊匹配，其余用等号 —— 对齐 RuoYi
// 前端表单的查询语义。
func {{.ClassNameLower}}ListQuery(q *query.{{.ClassName}}List) *gorm.DB {
	tx := db.DB.Table("{{.TableName}}")
{{range .QueryColumns}}	if q.{{.GoField}} != {{.GoZero}} {
{{if and .IsStringType (eq .QueryType "LIKE")}}		tx = tx.Where("{{.ColumnName}} LIKE ?", "%"+q.{{.GoField}}+"%")
{{else}}		tx = tx.Where("{{.ColumnName}} = ?", q.{{.GoField}})
{{end}}	}
{{end}}	return tx
}

// Select{{.ClassName}}List 分页查询{{.FunctionName}}。
func Select{{.ClassName}}List(q *query.{{.ClassName}}List, p *page.PageDomain) ([]entity.{{.ClassName}}, int64, error) {
	var rows []entity.{{.ClassName}}
	total, err := page.Paginate({{.ClassNameLower}}ListQuery(q), p, "", "", &rows)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// Select{{.ClassName}}ByID 按主键查询；查不到返回 (nil, nil)，不报错。
func Select{{.ClassName}}ByID(id {{.Pk.GoType}}) (*entity.{{.ClassName}}, error) {
	var x entity.{{.ClassName}}
	err := db.DB.Where("{{.Pk.ColumnName}} = ?", id).First(&x).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &x, nil
}

// Insert{{.ClassName}} 新增。
func Insert{{.ClassName}}(x *entity.{{.ClassName}}) error {
	return db.DB.Create(x).Error
}

// Update{{.ClassName}} 修改：只更新非零值字段。
//
// 对齐 RuoYi mapper.xml 里的 <set> 动态列 —— 传空值表示"这一列不改"，
// 而不是"把这一列改成空"。
func Update{{.ClassName}}(x *entity.{{.ClassName}}) error {
	sets := map[string]any{}
{{range .UpdateColumns}}{{if not .IsSuperColumn}}	if x.{{.GoField}} != {{.GoZero}} {
		sets["{{.ColumnName}}"] = x.{{.GoField}}
	}
{{end}}{{end}}	if len(sets) == 0 {
		return nil
	}
	return db.DB.Table("{{.TableName}}").Where("{{.Pk.ColumnName}} = ?", x.{{.Pk.GoField}}).Updates(sets).Error
}

// Delete{{.ClassName}}ByIDs 批量删除（物理删除）。
//
// 若业务上是逻辑删除（del_flag='2'），把这里改成 Updates(map[string]any{"del_flag": "2"})。
func Delete{{.ClassName}}ByIDs(ids []{{.Pk.GoType}}) error {
	if len(ids) == 0 {
		return nil
	}
	return db.DB.Table("{{.TableName}}").Where("{{.Pk.ColumnName}} IN ?", ids).Delete(nil).Error
}
`,

	"service.go": `package service

import (
	"{{.GoModule}}/internal/model/entity"
	"{{.GoModule}}/internal/model/query"
	"{{.GoModule}}/internal/repository"
	"{{.GoModule}}/pkg/page"
)

// Select{{.ClassName}}List {{.FunctionName}}列表。
func Select{{.ClassName}}List(q *query.{{.ClassName}}List, p *page.PageDomain) ([]entity.{{.ClassName}}, int64, error) {
	return repository.Select{{.ClassName}}List(q, p)
}

// Select{{.ClassName}}ByID {{.FunctionName}}详情。
//
// 唯一性/存在性校验请加在这里（参考 InsertPost 的 CheckPostNameUnique）：
// repository 只管查库，业务规则属于 service。
func Select{{.ClassName}}ByID(id {{.Pk.GoType}}) (*entity.{{.ClassName}}, error) {
	return repository.Select{{.ClassName}}ByID(id)
}

// Insert{{.ClassName}} 新增{{.FunctionName}}。
func Insert{{.ClassName}}(x *entity.{{.ClassName}}) error {
	return repository.Insert{{.ClassName}}(x)
}

// Update{{.ClassName}} 修改{{.FunctionName}}。
func Update{{.ClassName}}(x *entity.{{.ClassName}}) error {
	return repository.Update{{.ClassName}}(x)
}

// Delete{{.ClassName}}ByIDs 删除{{.FunctionName}}。
func Delete{{.ClassName}}ByIDs(ids []{{.Pk.GoType}}) error {
	return repository.Delete{{.ClassName}}ByIDs(ids)
}
`,

	"handler.go": `package system

import (
	"net/http"
{{if eq .Pk.GoType "int64"}}	"strconv"
{{else}}	"strings"
{{end}}
	"{{.GoModule}}/internal/model/entity"
	"{{.GoModule}}/internal/model/query"
	"{{.GoModule}}/internal/service"
	"{{.GoModule}}/pkg/page"
	"{{.GoModule}}/pkg/result"

	"github.com/gin-gonic/gin"
)

// {{.ClassName}}Handler {{.FunctionName}}，权限串前缀 {{.PermissionPrefix}}。
//
// 权限**不在这里判断**：本项目由 internal/router/perm_table.go 登记、
// middleware.Permi 统一校验。要加的条目见同批生成的 routes.txt。
type {{.ClassName}}Handler struct{}

// List GET {{.PathPrefix}}/list
func ({{.ClassName}}Handler) List(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.{{.ClassName}}List
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	rows, total, err := service.Select{{.ClassName}}List(&q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.{{.ClassName}}{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// GetInfo GET {{.PathPrefix}}/:{{.Pk.JavaField}}
func ({{.ClassName}}Handler) GetInfo(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
{{if eq .Pk.GoType "int64"}}	id, err := strconv.ParseInt(c.Param("{{.Pk.JavaField}}"), 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
{{else}}	id := c.Param("{{.Pk.JavaField}}")
{{end}}	x, err := service.Select{{.ClassName}}ByID(id)
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(x))
}

// Add POST {{.PathPrefix}}
func ({{.ClassName}}Handler) Add(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var x entity.{{.ClassName}}
	if err := c.ShouldBindJSON(&x); err != nil {
		fail(c, err)
		return
	}
	if err := service.Insert{{.ClassName}}(&x); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// Edit PUT {{.PathPrefix}}
func ({{.ClassName}}Handler) Edit(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var x entity.{{.ClassName}}
	if err := c.ShouldBindJSON(&x); err != nil {
		fail(c, err)
		return
	}
	if err := service.Update{{.ClassName}}(&x); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// Remove DELETE {{.PathPrefix}}/:{{.Pk.JavaField}}s
func ({{.ClassName}}Handler) Remove(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
{{if eq .Pk.GoType "int64"}}	ids := parseIDs(c.Param("{{.Pk.JavaField}}s"))
{{else}}	ids := strings.Split(c.Param("{{.Pk.JavaField}}s"), ",")
{{end}}	if len(ids) == 0 {
		result.Fail(c, result.Error("未选择要删除的{{.FunctionName}}"))
		return
	}
	if err := service.Delete{{.ClassName}}ByIDs(ids); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}
`,

	"routes.txt": `接入步骤（三步，缺一步接口就是 404 或 401）

1) internal/router/perm_constants.go（或 perm_table.go 的常量区）加权限常量：

	Perm{{.ModuleNameCap}}{{.ClassNameCap}}List   = "{{.PermissionPrefix}}:list"
	Perm{{.ModuleNameCap}}{{.ClassNameCap}}Query  = "{{.PermissionPrefix}}:query"
	Perm{{.ModuleNameCap}}{{.ClassNameCap}}Add    = "{{.PermissionPrefix}}:add"
	Perm{{.ModuleNameCap}}{{.ClassNameCap}}Edit   = "{{.PermissionPrefix}}:edit"
	Perm{{.ModuleNameCap}}{{.ClassNameCap}}Remove = "{{.PermissionPrefix}}:remove"

2) internal/router/perm_table.go 的 permRoutes 加条目（顺序随意，重复会被测试拦下）：

	{"GET",    "{{.PathPrefix}}/list",         Perm{{.ModuleNameCap}}{{.ClassNameCap}}List,   "{{.ClassName}}Handler"},
	{"GET",    "{{.PathPrefix}}/:{{.Pk.JavaField}}", Perm{{.ModuleNameCap}}{{.ClassNameCap}}Query,  "{{.ClassName}}Handler"},
	{"POST",   "{{.PathPrefix}}",             Perm{{.ModuleNameCap}}{{.ClassNameCap}}Add,    "{{.ClassName}}Handler"},
	{"PUT",    "{{.PathPrefix}}",             Perm{{.ModuleNameCap}}{{.ClassNameCap}}Edit,   "{{.ClassName}}Handler"},
	{"DELETE", "{{.PathPrefix}}/:{{.Pk.JavaField}}s", Perm{{.ModuleNameCap}}{{.ClassNameCap}}Remove, "{{.ClassName}}Handler"},

3) internal/router/router.go 注册路由（perm() 会自动挂 JWTAuth + Permi）：

	{{.ClassNameLower}} := system.{{.ClassName}}Handler{}
	perm(r, "GET",    "{{.PathPrefix}}/list", {{.ClassNameLower}}.List)
	perm(r, "GET",    "{{.PathPrefix}}/:{{.Pk.JavaField}}", {{.ClassNameLower}}.GetInfo)
	perm(r, "POST",   "{{.PathPrefix}}", {{.ClassNameLower}}.Add)
	perm(r, "PUT",    "{{.PathPrefix}}", {{.ClassNameLower}}.Edit)
	perm(r, "DELETE", "{{.PathPrefix}}/:{{.Pk.JavaField}}s", {{.ClassNameLower}}.Remove)

4) 执行同批生成的 sql/{{.TableName}}.sql，把菜单与按钮权限插进 sys_menu
   （admin 角色默认全通过，其它角色要在"角色管理"里勾选）。

注意：加完 1)+2) 后跑 go test ./internal/router/ —— perm_diff_test.go 会检查
Go 侧登记的路由是否与原 Java 一致。生成的模块 Java 侧没有对应控制器，
所以要把新条目加进测试的"仅 Go 侧"白名单（whitelist），否则会被判成"多余"。
`,

	"sql.sql": `-- 菜单 SQL
insert into sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, update_by, update_time, remark)
values('{{.FunctionName}}', '{{.ParentMenuId}}', '1', '{{.BusinessName}}', '{{.ModuleName}}/{{.BusinessName}}/index', 1, 0, 'C', '0', '0', '{{.PermissionPrefix}}:list', '#', 'admin', sysdate(), '', null, '{{.FunctionName}}菜单');

-- 按钮父菜单ID
SELECT @parentId := LAST_INSERT_ID();

-- 按钮 SQL
insert into sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, update_by, update_time, remark)
values('{{.FunctionName}}查询', @parentId, '1',  '#', '', 1, 0, 'F', '0', '0', '{{.PermissionPrefix}}:query',        '#', 'admin', sysdate(), '', null, '');

insert into sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, update_by, update_time, remark)
values('{{.FunctionName}}新增', @parentId, '2',  '#', '', 1, 0, 'F', '0', '0', '{{.PermissionPrefix}}:add',          '#', 'admin', sysdate(), '', null, '');

insert into sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, update_by, update_time, remark)
values('{{.FunctionName}}修改', @parentId, '3',  '#', '', 1, 0, 'F', '0', '0', '{{.PermissionPrefix}}:edit',         '#', 'admin', sysdate(), '', null, '');

insert into sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, update_by, update_time, remark)
values('{{.FunctionName}}删除', @parentId, '4',  '#', '', 1, 0, 'F', '0', '0', '{{.PermissionPrefix}}:remove',       '#', 'admin', sysdate(), '', null, '');

insert into sys_menu (menu_name, parent_id, order_num, path, component, is_frame, is_cache, menu_type, visible, status, perms, icon, create_by, create_time, update_by, update_time, remark)
values('{{.FunctionName}}导出', @parentId, '5',  '#', '', 1, 0, 'F', '0', '0', '{{.PermissionPrefix}}:export',       '#', 'admin', sysdate(), '', null, '');
`,
}

// Parse 解析全部模板。
func Parse() (map[string]*template.Template, error) {
	out := make(map[string]*template.Template, len(raw))
	for name, body := range raw {
		t, err := template.New(name).Funcs(funcs).Parse(body)
		if err != nil {
			return nil, err
		}
		out[name] = t
	}
	return out, nil
}

var funcs = template.FuncMap{
	"lower": strings.ToLower,
	// tick 输出一个反引号。
	//
	// 模板本体是 Go 的**原始字符串**（反引号包住），里面就不能再出现反引号，
	// 而生成的 Go 代码到处是结构体标签。与其把模板拆成一段段拼接，
	// 不如让模板在渲染时输出它。
	"tick": func() string { return "`" },
}

// ParsePath 输出路径本身也是模板（含 {{.ClassName}} 之类），单独解析。
func ParsePath(path string) (*template.Template, error) {
	return template.New("path").Parse(path)
}
