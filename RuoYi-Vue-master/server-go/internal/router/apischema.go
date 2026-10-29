package router

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
)

// 本文件给 /v3/api-docs 补上请求体与查询参数的 schema，
// 让 Swagger UI 能展开字段、Try it out 能直接填参数。
//
// DIFF(review) —— 为什么是"注册表 + 反射"而不是手写 JSON schema：
//
// Java 侧 springdoc 是扫 @Schema / @RequestBody 注解从**代码**生成的，
// 字段改动会自动反映到文档里。Go 侧没有这套注解，如果这里手写一份
// JSON schema，字段一改就漂移，而且没人会发现。
// 所以做成：注册表只声明"这条路由用哪个 Go 结构体"，字段清单、类型、
// 必填、说明全部**从结构体本身反射**出来。改结构体，文档自动跟着改。
//
// 字段说明取自 **MySQL 的列注释**（information_schema.COLUMNS.COLUMN_COMMENT），
// 而不是另写一份中文对照——列注释是 RuoYi 自带、随建表 SQL 走的，不会漂移；
// 取不到就留空，不影响字段本身。

// apiSchema 一条路由的请求/查询参数定义。
type apiSchema struct {
	// Body 请求体对应的 Go 结构体（POST/PUT）。为 nil 表示无请求体。
	Body any
	// Query 查询参数对应的 Go 结构体（GET 列表）。为 nil 表示无查询参数。
	QueryStr any
	// Required 必填字段的 **json 名**。
	//
	// 来源不是反射（Go 结构体上没有 binding:"required" 之类的标记），
	// 而是 Java 侧 @NotNull/@NotBlank/@Xss 的校验注解，逐个抄过来。
	// 留空表示全部可选。
	Required []string
	// Extra 结构体之外、接口确实还接收的字段。
	//
	// 典型是 password：SysUser 上它是 json:"-"（绝不从详情接口泄漏散列），
	// 新增/重置密码靠 handler 里一个独立字段接收（见 user_handler.go 的 Add），
	// 所以它不在反射结果里，只能在这里补上。
	Extra []extraField
	// Summary 接口说明，覆盖默认的权限串。
	Summary string
}

// extraField 一个额外字段。
type extraField struct {
	Name string
	Type string // string / integer / boolean
	Desc string
}

// apiSchemas 高频接口的 schema 注册表。
//
// key 必须与 perm_table.go 里的 {"Method", "Path"} 完全一致，
// 否则注册不上去（buildOpenAPI 会静默跳过 —— 见下面的校验测试）。
//
// 只登记增删改查这类**前端表单直接对应实体**的接口；
// 没登记的仍然只有路径和权限串（Swagger 里点开没有参数，但不会错）。
var apiSchemas = map[string]apiSchema{
	// ---------------------------------------------------------------- 用户
	"POST /system/user": {
		Body:     entity.SysUser{},
		Required: []string{"userName", "nickName", "password"},
		Extra: []extraField{
			{Name: "password", Type: "string", Desc: "密码（明文，后端 BCrypt 加密后存储）"},
		},
		Summary: "新增用户",
	},
	"PUT /system/user": {
		Body:     entity.SysUser{},
		Required: []string{"userId", "userName", "nickName"},
		Summary:  "修改用户",
	},
	"PUT /system/user/resetPwd": {
		Required: []string{"userId", "password"},
		Extra: []extraField{
			{Name: "userId", Type: "integer", Desc: "用户ID"},
			{Name: "password", Type: "string", Desc: "新密码（明文）"},
		},
		Summary: "重置用户密码（管理员）",
	},
	"PUT /system/user/changeStatus": {
		Required: []string{"userId", "status"},
		Extra: []extraField{
			{Name: "userId", Type: "integer", Desc: "用户ID"},
			{Name: "status", Type: "string", Desc: "状态：0 正常、1 停用"},
		},
		Summary: "修改用户状态",
	},
	"PUT /system/user/profile/updatePwd": {
		Required: []string{"oldPassword", "newPassword"},
		Extra: []extraField{
			{Name: "oldPassword", Type: "string", Desc: "原密码"},
			{Name: "newPassword", Type: "string", Desc: "新密码"},
		},
		Summary: "个人中心修改密码",
	},
	"GET /system/user/list": {QueryStr: query.UserList{}, Summary: "查询用户列表"},
	// ---------------------------------------------------------------- 角色
	"POST /system/role": {
		Body:     entity.SysRole{},
		Required: []string{"roleName", "roleKey"},
		Summary:  "新增角色",
	},
	"PUT /system/role": {
		Body:     entity.SysRole{},
		Required: []string{"roleId", "roleName", "roleKey"},
		Summary:  "修改角色",
	},
	"GET /system/role/list": {QueryStr: query.RoleList{}, Summary: "查询角色列表"},
	// ---------------------------------------------------------------- 部门
	"POST /system/dept": {
		Body:     entity.SysDept{},
		Required: []string{"parentId", "deptName"},
		Summary:  "新增部门",
	},
	"PUT /system/dept": {
		Body:     entity.SysDept{},
		Required: []string{"deptId", "parentId", "deptName"},
		Summary:  "修改部门",
	},
	"GET /system/dept/list": {QueryStr: query.DeptList{}, Summary: "查询部门列表"},
	// ---------------------------------------------------------------- 岗位
	"POST /system/post": {
		Body:     entity.SysPost{},
		Required: []string{"postName", "postCode"},
		Summary:  "新增岗位",
	},
	"PUT /system/post": {
		Body:     entity.SysPost{},
		Required: []string{"postId", "postName", "postCode"},
		Summary:  "修改岗位",
	},
	"GET /system/post/list": {QueryStr: query.PostList{}, Summary: "查询岗位列表"},
	// ---------------------------------------------------------------- 字典
	"POST /system/dict/type": {
		Body:     entity.SysDictType{},
		Required: []string{"dictName", "dictType"},
		Summary:  "新增字典类型",
	},
	"PUT /system/dict/type": {
		Body:     entity.SysDictType{},
		Required: []string{"dictId", "dictName", "dictType"},
		Summary:  "修改字典类型",
	},
	"GET /system/dict/type/list": {QueryStr: query.DictTypeList{}, Summary: "查询字典类型列表"},
	"POST /system/dict/data": {
		Body:     entity.SysDictData{},
		Required: []string{"dictType", "dictLabel", "dictValue"},
		Summary:  "新增字典数据",
	},
	"PUT /system/dict/data": {
		Body:     entity.SysDictData{},
		Required: []string{"dictCode", "dictType", "dictLabel", "dictValue"},
		Summary:  "修改字典数据",
	},
	"GET /system/dict/data/list": {QueryStr: query.DictDataList{}, Summary: "查询字典数据列表"},
	// ---------------------------------------------------------------- 参数 / 通知 / 菜单
	"POST /system/config": {
		Body:     entity.SysConfig{},
		Required: []string{"configName", "configKey", "configValue"},
		Summary:  "新增参数",
	},
	"PUT /system/config": {
		Body:     entity.SysConfig{},
		Required: []string{"configId", "configName", "configKey", "configValue"},
		Summary:  "修改参数",
	},
	"GET /system/config/list": {QueryStr: query.ConfigList{}, Summary: "查询参数列表"},
	"POST /system/notice": {
		Body:     entity.SysNotice{},
		Required: []string{"noticeTitle", "noticeType"},
		Summary:  "新增通知公告",
	},
	"PUT /system/notice": {
		Body:     entity.SysNotice{},
		Required: []string{"noticeId", "noticeTitle", "noticeType"},
		Summary:  "修改通知公告",
	},
	"GET /system/notice/list": {QueryStr: query.NoticeList{}, Summary: "查询通知公告列表"},
	"POST /system/menu": {
		Body:     entity.SysMenu{},
		Required: []string{"menuName"},
		Summary:  "新增菜单",
	},
	"PUT /system/menu": {
		Body:     entity.SysMenu{},
		Required: []string{"menuId", "menuName"},
		Summary:  "修改菜单",
	},
	"GET /system/menu/list": {QueryStr: query.MenuList{}, Summary: "查询菜单列表"},
}

// modelTables 结构体 -> 表名，用于取列注释。
var modelTables = map[string]string{
	"SysUser":     "sys_user",
	"SysRole":     "sys_role",
	"SysDept":     "sys_dept",
	"SysPost":     "sys_post",
	"SysDictType": "sys_dict_type",
	"SysDictData": "sys_dict_data",
	"SysConfig":   "sys_config",
	"SysNotice":   "sys_notice",
	"SysMenu":     "sys_menu",
}

// ---------------------------------------------------------------- 反射生成 schema

// reGormColumn 从 `gorm:"column:xxx;..."` 里取出列名。
var reGormColumn = regexp.MustCompile(`column:([^;]+)`)

// schemaName Go 类型名即组件名（SysUser -> #/components/schemas/SysUser）。
func schemaName(t reflect.Type) string {
	return t.Name()
}

// structSchema 把一个结构体反射成 OpenAPI 的 object schema。
//
// 处理规则：
//   - json:"-" 直接跳过（Go 不序列化的，文档里也不该出现，
//     例如 SysUser.Password —— 那条注释里写明了为什么 json:"-"）
//   - 匿名字段（嵌入的 BaseModel）**展开**，与 Go 的序列化行为一致
//   - 关联对象（Dept / Roles）与 map 不展开，只标成 object / array，
//     否则一个 SysUser 会把整个库的关系都带进来
func structSchema(t reflect.Type) map[string]any {
	comments := map[string]string{}
	if table, ok := modelTables[t.Name()]; ok {
		comments = columnComments(table)
	}
	props := map[string]any{}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous {
			// 嵌入字段：把它的字段并入当前 schema（与 encoding/json 一致）
			sub := structSchema(f.Type)
			if subProps, ok := sub["properties"].(map[string]any); ok {
				for k, v := range subProps {
					props[k] = v
				}
			}
			continue
		}
		name, skip := jsonName(f)
		if skip {
			continue
		}
		prop := typeSchema(f.Type)
		m := reGormColumn.FindStringSubmatch(f.Tag.Get("gorm"))
		if len(m) == 2 {
			if c := comments[m[1]]; c != "" {
				prop["description"] = c
			}
		}
		props[name] = prop
	}
	return map[string]any{"type": "object", "properties": props}
}

// jsonName 取字段的 json 名；第二个返回值为 true 表示跳过。
func jsonName(f reflect.StructField) (string, bool) {
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", true
	}
	name := f.Name
	if i := strings.Index(tag, ","); i >= 0 {
		name = tag[:i]
	} else if tag != "" {
		name = tag
	}
	if name == "" {
		name = f.Name
	}
	return name, false
}

// typeSchema Go 类型 -> OpenAPI 类型。
func typeSchema(t reflect.Type) map[string]any {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer", "format": "int64"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": typeSchema(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": true}
	case reflect.Struct:
		// time.Time 是特例：JSON 里是字符串
		if t.Name() == "Time" {
			return map[string]any{"type": "string", "format": "date-time"}
		}
		// 其余结构体（关联对象）不展开，避免把整个关系图拉进来
		return map[string]any{"type": "object"}
	default:
		return map[string]any{"type": "string"}
	}
}

// ---------------------------------------------------------------- 列注释

var (
	commentMu sync.Mutex
	comments  = map[string]map[string]string{} // 表名 -> 列名 -> 注释
)

// columnComments 取表的列注释，结果缓存。
//
// 失败（库没连上、表不存在）就返回空 map —— **字段说明只是锦上添花**，
// 绝不能因为取不到注释就让 /v3/api-docs 挂掉。
func columnComments(table string) map[string]string {
	commentMu.Lock()
	defer commentMu.Unlock()
	if c, ok := comments[table]; ok {
		return c
	}
	c := map[string]string{}
	if db.DB != nil {
		var rows []struct {
			Name    string `gorm:"column:COLUMN_NAME"`
			Comment string `gorm:"column:COLUMN_COMMENT"`
		}
		err := db.DB.Raw(
			"SELECT COLUMN_NAME, COLUMN_COMMENT FROM information_schema.COLUMNS "+
				"WHERE table_schema = DATABASE() AND table_name = ?", table,
		).Scan(&rows).Error
		if err == nil {
			for _, r := range rows {
				c[r.Name] = r.Comment
			}
		}
	}
	comments[table] = c
	return c
}

// ---------------------------------------------------------------- 装配

// buildComponents 生成 components.schemas：为每个用到的实体建一个命名 schema。
func buildComponents() map[string]any {
	out := map[string]any{}
	seen := map[string]bool{}
	for _, s := range apiSchemas {
		if s.Body == nil {
			continue
		}
		t := indirect(reflect.TypeOf(s.Body))
		name := schemaName(t)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out[name] = structSchema(t)
	}
	// 统一的响应包装，与 pkg/result 的返回结构一致
	out["AjaxResult"] = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code": map[string]any{"type": "integer", "description": "状态码，200 成功"},
			"msg":  map[string]any{"type": "string", "description": "返回消息"},
			"data": map[string]any{"type": "object", "description": "返回数据"},
		},
	}
	out["TableResult"] = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code":  map[string]any{"type": "integer"},
			"msg":   map[string]any{"type": "string"},
			"rows":  map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
			"total": map[string]any{"type": "integer", "description": "总记录数"},
		},
	}
	return out
}

func indirect(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// applySchema 把注册表里的定义挂到某个 operation 上。
// 返回是否命中注册表。
func applySchema(op map[string]any, method, path string) bool {
	s, ok := apiSchemas[method+" "+path]
	if !ok {
		return false
	}
	if s.Summary != "" {
		op["summary"] = s.Summary
	}
	if s.Body != nil || len(s.Extra) > 0 {
		// 请求体 schema **内联**而不是 $ref：同一个结构体（例如 SysUser）
		// 被 POST 和 PUT 共用，但两者的 required 不同（新增要 password、
		// 修改要 userId），抽成命名组件就只能留一份 required，必然有一边是错的。
		sch := map[string]any{"type": "object", "properties": map[string]any{}}
		props := sch["properties"].(map[string]any)
		if s.Body != nil {
			from := structSchema(indirect(reflect.TypeOf(s.Body)))
			if fp, ok := from["properties"].(map[string]any); ok {
				for k, v := range fp {
					props[k] = v
				}
			}
		}
		for _, e := range s.Extra {
			typ := e.Type
			if typ == "" {
				typ = "string"
			}
			p := map[string]any{"type": typ}
			if e.Desc != "" {
				p["description"] = e.Desc
			}
			props[e.Name] = p
		}
		// required 只保留 schema 里真实存在的字段，避免写错名字污染文档
		req := []string{}
		for _, r := range s.Required {
			if _, ok := props[r]; ok {
				req = append(req, r)
			}
		}
		if len(req) > 0 {
			sch["required"] = req
		}
		op["requestBody"] = map[string]any{
			"required": true,
			"content": map[string]any{
				"application/json": map[string]any{"schema": sch},
			},
		}
	}
	if s.QueryStr != nil {
		t := indirect(reflect.TypeOf(s.QueryStr))
		sch := structSchema(t)
		params := []any{}
		if props, ok := sch["properties"].(map[string]any); ok {
			// map 遍历顺序不定，先按名字排序，文档才稳定
			names := make([]string, 0, len(props))
			for n := range props {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				p := map[string]any{
					"name":     n,
					"in":       "query",
					"required": false,
					"schema":   props[n],
				}
				params = append(params, p)
			}
		}
		// 分页与排序参数不在查询结构体里 —— pkg/page 是直接按
		// pageNum/pageSize/orderBy/isAsc 这几个 key 读 query 的
		// （见 pkg/page/page.go 的 KeyPageNum 等常量），反射不到，这里补上。
		for _, q := range []struct{ name, desc string }{
			{page.KeyPageNum, "页码，默认 1"},
			{page.KeyPageSize, "每页条数，默认 10"},
			{page.KeyOrderByColumn, "排序字段"},
			{page.KeyIsAsc, "排序方向：asc / desc"},
		} {
			params = append(params, map[string]any{
				"name":        q.name,
				"in":          "query",
				"required":    false,
				"description": q.desc,
				"schema":      map[string]any{"type": "string"},
			})
		}
		// RuoYi 前端的日期范围是 params[beginTime] / params[endTime]
		// （gin 不认这种方括号嵌套 key，query 包里是手工取的，见 FillDateRange）
		for _, k := range []string{query.KeyBeginTime, query.KeyEndTime} {
			params = append(params, map[string]any{
				"name":        k,
				"in":          "query",
				"required":    false,
				"description": "日期范围（与前端一致用方括号 key）",
				"schema":      map[string]any{"type": "string", "format": "date-time"},
			})
		}
		if len(params) > 0 {
			op["parameters"] = append(asSlice(op["parameters"]), params...)
		}
	}
	// 响应：列表接口给 TableResult，其余给 AjaxResult
	if _, ok := op["requestBody"]; ok {
		op["responses"].(map[string]any)["200"] = map[string]any{
			"description": "成功",
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{"$ref": "#/components/schemas/AjaxResult"},
				},
			},
		}
	} else if method == "GET" {
		op["responses"].(map[string]any)["200"] = map[string]any{
			"description": "成功",
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{"$ref": "#/components/schemas/TableResult"},
				},
			},
		}
	}
	return true
}

// asSlice 把可能不存在的 parameters 转成切片。
func asSlice(v any) []any {
	if v == nil {
		return nil
	}
	s, ok := v.([]any)
	if !ok {
		return nil
	}
	return s
}
