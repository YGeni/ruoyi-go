package system

import "html/template"

// druidTpl 数据监控页面的 HTML。
//
// 用 html/template 是为了自动转义（库名、版本这些来自数据库/Redis 的字符串
// 不能直接拼进 HTML）。页面 10 秒自动刷新一次，与 RuoYi 前端那边
// 服务监控的刷新节奏一致。
var druidTpl = template.Must(template.New("druid").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta http-equiv="refresh" content="10">
<title>{{.Title}} - RuoYi-Go</title>
<style>
  body{margin:0;padding:24px;font:14px/1.6 -apple-system,BlinkMacSystemFont,"PingFang SC","Microsoft YaHei",sans-serif;background:#f5f7fa;color:#303133}
  h1{font-size:20px;margin:0 0 4px}
  .meta{color:#909399;font-size:12px;margin-bottom:20px}
  .note{background:#fdf6ec;border-left:4px solid #e6a23c;padding:10px 14px;margin-bottom:20px;color:#a2691d;font-size:13px}
  .card{background:#fff;border:1px solid #e4e7ed;border-radius:4px;margin-bottom:16px;overflow:hidden}
  .card h2{margin:0;padding:12px 16px;font-size:15px;border-bottom:1px solid #e4e7ed;background:#fafafa}
  table{width:100%;border-collapse:collapse}
  td{padding:8px 16px;border-bottom:1px solid #f2f6fc}
  tr:last-child td{border-bottom:none}
  td:first-child{width:220px;color:#606266}
  td:last-child{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}
</style>
</head>
<body>
<h1>{{.Title}}</h1>
<div class="meta">更新于 {{.Updated}} · 已运行 {{.Uptime}} · 每 10 秒自动刷新</div>
<div class="note">{{.Note}}</div>
{{range .Sections}}
<div class="card">
  <h2>{{.Name}}</h2>
  <table>
    {{range .Rows}}<tr><td>{{.K}}</td><td>{{.V}}</td></tr>
    {{end}}
  </table>
</div>
{{end}}
</body>
</html>`))
