package router

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/YGeni/ruoyi-go/server-go/assets"

	"github.com/gin-gonic/gin"
)

// swaggerUIIndex 真 Swagger UI 的页面骨架。
//
// 用的是 swagger-ui-dist 的 StandaloneLayout（顶栏 + Authorize 按钮），
// 但**不是**它的默认 index.html，因为有两件事必须自己处理：
//
//  1. 接口文档的地址要带登录态。iframe 发不出 Authorization 头，
//     所以前端是把 token 拼在 ?token= 上的；这里把 token 读出来，
//     既用于取 /v3/api-docs，也通过 requestInterceptor 注入到
//     每个 try-it-out 请求里，否则点"执行"全是 401。
//
//  2. 请求前缀要自己算。页面可能经前端代理访问（/dev-api/swagger-ui/index.html），
//     也可能直连后端（/swagger-ui/index.html）；而 /v3/api-docs 里的
//     path 都是 /system/user/list 这种绝对路径，不指定 servers 的话
//     会被拼到 /dev-api/swagger-ui/ 下面去，全部 404。
//     所以从 location.pathname 反推出前缀，写进 doc.servers。
const swaggerUIIndex = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>系统接口 - 天才管理系统</title>
<link rel="stylesheet" href="./swagger-ui.css">
<link rel="icon" type="image/png" href="./favicon-32x32.png" sizes="32x32">
<link rel="icon" type="image/png" href="./favicon-16x16.png" sizes="16x16">
<style>
  html { box-sizing: border-box; overflow-y: scroll; height: 100%; }
  body { margin: 0; }
  .topbar { display: none; }
</style>
</head>
<body>
<div id="swagger-ui"></div>
<script src="./swagger-ui-bundle.js"></script>
<script src="./swagger-ui-standalone-preset.js"></script>
<script>
window.onload = async function () {
  var token = new URLSearchParams(location.search).get('token') || ''

  // 反推前缀：/dev-api/swagger-ui/index.html -> /dev-api/ ；
  //            /swagger-ui/index.html        -> /
  //            /swagger-ui.html              -> /
  var prefix = location.pathname
    .replace(/swagger-ui\/index\.html$/, '')
    .replace(/swagger-ui\.html$/, '')
  var docUrl = prefix + 'v3/api-docs' + (token ? '?token=' + encodeURIComponent(token) : '')

  var doc
  try {
    var res = await fetch(docUrl)
    doc = await res.json()
  } catch (e) {
    document.getElementById('swagger-ui').innerHTML =
      '<div style="padding:24px;font:14px sans-serif;color:#a2691d;background:#fdf6ec;' +
      'border-left:4px solid #e6a23c">接口文档加载失败：' + (e && e.message ? e.message : e) +
      '。请确认登录态是否有效（会话过期后刷新页面即可）。</div>'
    return
  }

  // 关键：/v3/api-docs 里的 path 是 /system/user/list 这种绝对路径，
  // 必须告诉 swagger-ui 前缀，否则会拼到当前目录下去。
  if (!doc.servers || doc.servers.length === 0) {
    doc.servers = [{ url: prefix.replace(/\/$/, '') }]
  }

  SwaggerUIBundle({
    spec: doc,
    dom_id: '#swagger-ui',
    deepLinking: true,
    docExpansion: 'list',
    defaultModelsExpandDepth: -1,
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    plugins: [SwaggerUIBundle.plugins.DownloadUrl],
    layout: 'StandaloneLayout',
    // 把登录态注入每个 try-it-out 请求，否则点执行全是 401
    requestInterceptor: function (req) {
      if (token) { req.headers['Authorization'] = 'Bearer ' + token }
      return req
    }
  })
}
</script>
</body>
</html>`

// serveSwaggerUI 真正的 Swagger UI 页面。需要登录态（header 或 query）。
func serveSwaggerUI(c *gin.Context) {
	if !authForIframe(c) {
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUIIndex))
}

// serveSwaggerAsset 静态资源（css/js/png）。
//
// DIFF(review)：这些**不校验登录态**。iframe 取 css/js 时不带 token，
// 挡掉就是白屏；而这些文件里没有任何业务数据（真正的接口清单在
// /v3/api-docs，那个是校验的）。等价于 springdoc 的 webjar 静态资源也是公开的。
func serveSwaggerAsset(c *gin.Context, name string) {
	if strings.Contains(name, "..") || strings.Contains(name, "/") {
		c.Status(http.StatusNotFound)
		return
	}
	sub, err := fs.Sub(assets.SwaggerUI, "swagger-ui")
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.FileFromFS(name, http.FS(sub))
}
