package router

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
)

// TestAPISchemaKeysExist 注册表里的 key 必须在路由表里真实存在。
//
// apischema.go 的 key 是手写的字符串（"POST /system/user"），
// 拼错一个字 applySchema 就会**静默跳过** —— 文档里那个接口还是没有字段，
// 而且没有任何报错。这道测试就是防这个。
func TestAPISchemaKeysExist(t *testing.T) {
	known := map[string]bool{}
	for _, rp := range permRoutes {
		known[rp.Method+" "+rp.Path] = true
	}
	for _, rp := range permRoutesPhase2 {
		known[rp.Method+" "+rp.Path] = true
	}
	for k := range apiSchemas {
		if !known[k] {
			t.Errorf("apischema.go 里的 %q 在 perm_table.go 中不存在，key 写错了", k)
		}
	}
}

// TestAPISchemaRequiredFieldsReal Required 里写的字段名必须是结构体上真实存在的 json 名。
//
// 写错同样是静默失效（applySchema 会过滤掉），Swagger 里就看不到必填星号。
func TestAPISchemaRequiredFieldsReal(t *testing.T) {
	for key, s := range apiSchemas {
		if len(s.Required) == 0 {
			continue
		}
		props := map[string]any{}
		if s.Body != nil {
			if fp, ok := structSchema(indirect(reflect.TypeOf(s.Body)))["properties"].(map[string]any); ok {
				for k, v := range fp {
					props[k] = v
				}
			}
		}
		for _, e := range s.Extra {
			props[e.Name] = map[string]any{"type": e.Type}
		}
		for _, r := range s.Required {
			if _, ok := props[r]; !ok {
				t.Errorf("%s 的 Required 里的 %q 在请求体上不存在（json 名写错了？）", key, r)
			}
		}
	}
}

// TestOpenAPIDocBuilds 文档能生成，且高频接口真的带上了请求体/查询参数。
func TestOpenAPIDocBuilds(t *testing.T) {
	doc := buildOpenAPI()
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatal("paths 缺失")
	}
	// 请求体
	get := paths["/system/user"].(map[string]any)["post"].(map[string]any)
	if _, ok := get["requestBody"]; !ok {
		t.Fatal("POST /system/user 没有 requestBody")
	}
	// 查询参数
	list := paths["/system/user/list"].(map[string]any)["get"].(map[string]any)
	params, ok := list["parameters"].([]any)
	if !ok || len(params) == 0 {
		t.Fatal("GET /system/user/list 没有查询参数")
	}
	// 路径参数仍然在
	single := paths["/system/user/{userId}"].(map[string]any)["get"].(map[string]any)
	found := false
	for _, p := range single["parameters"].([]any) {
		if p.(map[string]any)["name"] == "userId" && p.(map[string]any)["in"] == "path" {
			found = true
		}
	}
	if !found {
		t.Error("GET /system/user/{userId} 丢了路径参数")
	}
	// 整体能序列化（map[string]any 里不能有 JSON 不支持的类型）
	if _, err := json.Marshal(doc); err != nil {
		t.Errorf("文档无法序列化：%v", err)
	}
}

// TestSchemaHidesPassword SysUser.Password 是 json:"-"，绝不能出现在文档里。
//
// 那条字段的注释里写明了原因：Go 侧不返回密码散列。文档同样不能泄露它。
func TestSchemaHidesPassword(t *testing.T) {
	sch := structSchema(indirect(reflect.TypeOf(entity.SysUser{})))
	props, ok := sch["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema 没有 properties")
	}
	if _, leaked := props["password"]; leaked {
		t.Error("password 出现在 schema 里")
	}
	// 顺带确认常用字段在，否则说明反射逻辑坏了
	for _, want := range []string{"userName", "nickName", "userId"} {
		if _, ok := props[want]; !ok {
			t.Errorf("schema 里少了 %s", want)
		}
	}
}
