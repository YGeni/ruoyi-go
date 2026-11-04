package router

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/middleware"

	"github.com/gin-gonic/gin"
)

// 本文件对应 Java 侧 springdoc-openapi 提供的地址，Go 侧自行实现：
//
//	GET /v3/api-docs               OpenAPI 3.0 描述文档
//	GET /swagger-ui/index.html      真 Swagger UI（swagger-ui-dist，资源 go:embed）
//	GET /swagger-ui/progress.html   迁移进度清单（自绘只读）
//
// 前端 src/views/tool/swagger/index.vue 用 iframe 指到 index.html；
// iframe 发不出 Authorization 头，token 走 ?token= 由 authForIframe 兜底。
//
// DIFF(review) —— 与 Java 侧 springdoc 的差别（有意的功能替换，非等价移植）：
//   - 数据源是 internal/router/perm_table.go —— 与 Java 控制器逐个注解对齐、
//     由 perm_diff_test.go 双向 diff 守护的路由表，路径/方法/权限串/控制器准确。
//   - Java 的 schema 由 springdoc 扫 @Schema / @RequestBody 注解生成；
//     Go 侧没有注解，改由 apischema.go 的「注册表 + 结构体反射」补上高频接口的
//     请求体、查询参数与响应体 schema（字段说明取 MySQL 列注释），
//     未登记的接口只有路径和权限串，没有字段级 schema。
//   - 页面分两套：index.html 是内嵌静态资源的真 Swagger UI，支持 Try it out，
//     requestInterceptor 把登录态注入每个请求；progress.html 是自绘只读清单，
//     额外标注「已实现 / 二期延后」，同时充当迁移进度视图。

// registered 记录 New() 实际注册到 gin 上的路由，用于标注「已实现」。
// New() 会被测试多次调用，所以每次整体重建而不是累加。
var registered = map[string]bool{}

// snapshotRoutes 在 New() 末尾调用，抓一份已注册路由的快照。
func snapshotRoutes(r *gin.Engine) {
	registered = map[string]bool{}
	for _, rt := range r.Routes() {
		registered[rt.Method+" "+rt.Path] = true
	}
}

// ---------------------------------------------------------------- /v3/api-docs

// OpenAPIDoc GET /v3/api-docs
//
// 返回 OpenAPI 3.0.3 文档。paths 来自 permRoutes + permRoutesPhase2，
// tags 按 Java 控制器分组，每条 operation 额外带上两个 RuoYi 特有的扩展字段：
//
//	x-ruoyi-permission  权限串（空串表示仅需登录态）
//	x-ruoyi-implemented 该端点在 Go 侧是否已注册（false = 二期延后）
//
// 鉴权：只读文档，但含全站接口清单，因此仍要求登录态。
// iframe 发不出 Authorization 头，与 /druid/login.html 一样
// 走 header 优先、query 兜底。
func OpenAPIDoc(c *gin.Context) {
	if !authForIframe(c) {
		return
	}
	doc := buildOpenAPI()
	c.Header("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(c.Writer)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
	}
}

// ---------------------------------------------------------------- Swagger UI

// SwaggerUIResource /swagger-ui/* 的统一入口。
//
// gin 的路由树不允许同一个位置既有静态段又有通配段，
// 所以 index.html、progress.html 和 css/js 全部走这一个 handler 分发，
// 而不是各注册一条路由。
func SwaggerUIResource(c *gin.Context) {
	switch c.Param("filepath") {
	case "", "/", "/index.html":
		serveSwaggerUI(c) // 真 Swagger UI（swagger-ui-dist）
		return
	case "/progress.html":
		serveSwaggerProgress(c) // 自绘清单，含"已实现 / 二期延后"
		return
	}
	serveSwaggerAsset(c, strings.TrimPrefix(c.Param("filepath"), "/"))
}

// serveSwaggerProgress 迁移进度视图。
//
// 真 Swagger UI 只认 OpenAPI 标准字段，看不出哪些端点在 Go 侧已实现，
// 所以自绘那份清单保留在 /swagger-ui/progress.html —— 它同时是迁移进度表。
func serveSwaggerProgress(c *gin.Context) {
	if !authForIframe(c) {
		return
	}
	var sb strings.Builder
	if err := swaggerTpl.Execute(&sb, buildSwaggerData()); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(sb.String()))
}

// authForIframe iframe 场景的登录态校验：header 优先、query 兜底。
//
// 与 internal/handler/system/monitor_druid.go 的做法一致；
// 理由同样写在 middleware.AuthWithToken 的注释里。
func authForIframe(c *gin.Context) bool {
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader(config.C.Token.Header)
		if i := strings.Index(token, " "); i > 0 { // 剥掉 "Bearer " 前缀
			token = token[i+1:]
		}
	}
	return middleware.AuthWithToken(c, token)
}

// ---------------------------------------------------------------- 数据构造

// apiEntry 给页面用的一行。
type apiEntry struct {
	Method      string
	Path        string // gin 风格（:id）
	OpenAPIPath string // OpenAPI 风格（{id}）
	Perm        string
	PermText    string
	Implemented bool
}

type apiGroup struct {
	Controller string
	Entries    []apiEntry
}

type swaggerData struct {
	Title       string
	AppName     string
	Note        string
	Total       int
	Implemented int
	Pending     int
	Groups      []apiGroup
	APIDocsURL  string
}

// normalizeEntryPath 给路径补上前导斜杠。
//
// Java 源码解析出的基准快照里，SysLoginController 的 getInfo / getRouters
// 两条注解没有前导斜杠（见 testdata/java_endpoints.json），permRoutesPhase2
// 照抄了它们。但 Go 侧实际注册的是 /getInfo、/getRouters：不规范化会导致
// 这两条已实现的端点被误标为「二期延后」，且 OpenAPI 里产出 "getInfo"
// 这种非法 path（规范要求以 / 开头）。
//
// 快照与交接清单保持 Java 原样（它们是 diff 基准），只在对外展示/匹配时规范化。
func normalizeEntryPath(p string) string {
	if p != "" && !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

// allEntries permRoutes（已实现/在册）+ permRoutesPhase2（二期延后）合成一份清单，
// 按 controller 分组、组内按 path 排序。
func allEntries() []apiEntry {
	var all []apiEntry
	add := func(list []RoutePerm) {
		for _, rp := range list {
			path := normalizeEntryPath(rp.Path)
			e := apiEntry{
				Method:      rp.Method,
				Path:        path,
				OpenAPIPath: toOpenAPIPath(path),
				Perm:        rp.Perm,
				Implemented: registered[rp.Method+" "+path],
			}
			if rp.Perm == "" {
				e.PermText = "（仅需登录）"
			} else {
				e.PermText = rp.Perm
			}
			all = append(all, e)
		}
	}
	add(permRoutes)
	add(permRoutesPhase2)
	sort.Slice(all, func(i, j int) bool {
		if all[i].Method != all[j].Method {
			return all[i].Method < all[j].Method
		}
		return all[i].Path < all[j].Path
	})
	return all
}

func groupByController(all []apiEntry) []apiGroup {
	m := map[string][]apiEntry{}
	var order []string
	for _, e := range all {
		ctrl := controllerOf(e)
		if _, ok := m[ctrl]; !ok {
			order = append(order, ctrl)
		}
		m[ctrl] = append(m[ctrl], e)
	}
	sort.Strings(order)
	groups := make([]apiGroup, 0, len(order))
	for _, c := range order {
		groups = append(groups, apiGroup{Controller: c, Entries: m[c]})
	}
	return groups
}

// controllerOf 反查条目所属的 Java 控制器。
func controllerOf(e apiEntry) string {
	for _, rp := range permRoutes {
		if rp.Method == e.Method && rp.Path == e.Path {
			return rp.Java
		}
	}
	for _, rp := range permRoutesPhase2 {
		if rp.Method == e.Method && rp.Path == e.Path {
			return rp.Java
		}
	}
	return "其他"
}

func buildSwaggerData() swaggerData {
	all := allEntries()
	implemented := 0
	for _, e := range all {
		if e.Implemented {
			implemented++
		}
	}
	return swaggerData{
		Title:   "系统接口",
		AppName: config.C.RuoYi.Name,
		Note: "本页是迁移进度清单，只读展示路由表：路径、方法、权限串、所属控制器" +
			"来自与 Java 控制器逐个注解对齐的 internal/router/perm_table.go" +
			"（由 perm_diff_test.go 双向 diff 守护）。字段级 schema 与 Try it out " +
			"请在 /swagger-ui/index.html（真 Swagger UI）查看。",
		Total:       len(all),
		Implemented: implemented,
		Pending:     len(all) - implemented,
		Groups:      groupByController(all),
		APIDocsURL:  "v3/api-docs",
	}
}

// ---------------------------------------------------------------- OpenAPI 文档

func buildOpenAPI() map[string]any {
	paths := map[string]any{}
	for _, e := range allEntries() {
		item, ok := paths[e.OpenAPIPath].(map[string]any)
		if !ok {
			item = map[string]any{}
			paths[e.OpenAPIPath] = item
		}
		op := map[string]any{
			"tags":        []string{controllerOf(e)},
			"summary":     e.PermText,
			"operationId": strings.ToLower(e.Method) + strings.ReplaceAll(e.OpenAPIPath, "/", "_"),
			"responses": map[string]any{
				"200": map[string]any{"description": "成功"},
				"401": map[string]any{"description": "未认证或会话已过期"},
			},
			// RuoYi 特有：权限串与 Go 侧实现状态
			"x-ruoyi-permission":  e.Perm,
			"x-ruoyi-implemented": e.Implemented,
		}
		if e.Perm != "" {
			op["security"] = []any{map[string]any{"BearerAuth": []string{}}}
		}
		if ps := pathParams(e.OpenAPIPath); len(ps) > 0 {
			params := make([]any, 0, len(ps))
			for _, p := range ps {
				params = append(params, map[string]any{
					"name":     p,
					"in":       "path",
					"required": true,
					"schema":   map[string]any{"type": "string"},
				})
			}
			op["parameters"] = params
		}
		// 挂上请求体 / 查询参数（见 apischema.go 的注册表）
		applySchema(op, e.Method, e.Path)
		item[strings.ToLower(e.Method)] = op
	}
	components := buildComponents()
	components["securitySchemes"] = map[string]any{
		"BearerAuth": map[string]any{
			"type":         "http",
			"scheme":       "bearer",
			"bearerFormat": "JWT",
		},
	}
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   config.C.RuoYi.Name + " API",
			"version": "1.0.0",
			"description": "由 Go 后端从路由表生成。路径/方法/权限串与 Java 控制器注解对齐；" +
				"高频接口的请求体、查询参数与响应体 schema 由「注册表 + Go 结构体反射」生成，" +
				"字段说明取自 MySQL 列注释（见 apischema.go）；未登记的接口没有字段级 schema。",
		},
		"components": components,
		"paths":      paths,
	}
}

// toOpenAPIPath gin 的 :id / *filepath 转成 OpenAPI 的 {id} / {filepath}。
func toOpenAPIPath(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if strings.HasPrefix(s, ":") {
			segs[i] = "{" + strings.TrimPrefix(s, ":") + "}"
		} else if strings.HasPrefix(s, "*") {
			segs[i] = "{" + strings.TrimPrefix(s, "*") + "}"
		}
	}
	return strings.Join(segs, "/")
}

// pathParams 抽出 {xxx} 形式的路径参数名。
func pathParams(p string) []string {
	var out []string
	for _, seg := range strings.Split(p, "/") {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			out = append(out, seg[1:len(seg)-1])
		}
	}
	return out
}
