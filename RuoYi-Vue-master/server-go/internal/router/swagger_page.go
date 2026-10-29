package router

import "html/template"

// swaggerTpl "系统接口"页面的 HTML。
//
// 与 druid 页面同理：html/template 自动转义（控制器名、路径、权限串都来自
// 代码里的表，虽然可信，但路径里出现 < > 会让页面串掉）。
// 不引 Swagger UI 的 CDN 资源 —— 离线就白屏，这里自绘只读列表 + 关键字过滤。
var swaggerTpl = template.Must(template.New("swagger").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<title>{{.Title}} - RuoYi-Go</title>
<style>
  body{margin:0;padding:24px;font:14px/1.6 -apple-system,BlinkMacSystemFont,"PingFang SC","Microsoft YaHei",sans-serif;background:#f5f7fa;color:#303133}
  h1{font-size:20px;margin:0 0 4px;display:inline-block}
  .meta{color:#909399;font-size:12px;margin:6px 0 16px}
  .note{background:#fdf6ec;border-left:4px solid #e6a23c;padding:10px 14px;margin-bottom:16px;color:#a2691d;font-size:13px}
  .bar{display:flex;align-items:center;gap:12px;margin-bottom:16px}
  input{flex:1;padding:7px 12px;border:1px solid #dcdfe6;border-radius:4px;font-size:14px;outline:none}
  input:focus{border-color:#409eff}
  .stat{font-size:12px;color:#606266;white-space:nowrap}
  .card{background:#fff;border:1px solid #e4e7ed;border-radius:4px;margin-bottom:14px;overflow:hidden}
  .card h2{margin:0;padding:10px 16px;font-size:14px;background:#fafafa;border-bottom:1px solid #e4e7ed;font-weight:600}
  table{width:100%;border-collapse:collapse}
  tr{border-bottom:1px solid #f2f6fc}
  tr:last-child{border-bottom:none}
  tr:hover{background:#f5f7fa}
  td{padding:6px 16px;vertical-align:middle}
  .m{display:inline-block;min-width:52px;text-align:center;padding:1px 7px;border-radius:3px;color:#fff;font-size:11px;font-weight:700}
  .GET{background:#61affe} .POST{background:#49cc90} .PUT{background:#fca130} .DELETE{background:#f93e3e}
  .p{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}
  .perm{color:#909399;font-size:12px}
  .tag{font-size:11px;padding:1px 6px;border-radius:10px;border:1px solid #dcdfe6;color:#909399}
  .done{color:#67c23a;border-color:#b3e19d}
  .todo{color:#e6a23c;border-color:#f5dab1}
  code{background:#f5f7fa;padding:2px 6px;border-radius:3px}
</style>
</head>
<body>
<h1>{{.Title}}</h1>
<div class="stat">共 {{.Total}} 个端点 · 已实现 {{.Implemented}} · 二期延后 {{.Pending}}</div>
<div class="meta">机器可读版本：<code>{{.APIDocsURL}}</code></div>
<div class="note">{{.Note}}</div>
<div class="bar">
  <input id="q" placeholder="按路径、权限串或控制器过滤，例如 user / system:user:list / SysUserController" oninput="filter()">
</div>
<div id="list">
{{range .Groups}}
{{$ctrl := .Controller}}
<div class="card" data-ctrl="{{$ctrl}}">
  <h2>{{$ctrl}} <span class="stat">（{{len .Entries}}）</span></h2>
  <table>
    {{range .Entries}}
    <tr data-k="{{.Method}} {{.Path}} {{.PermText}} {{$ctrl}}">
      <td style="width:70px"><span class="m {{.Method}}">{{.Method}}</span></td>
      <td style="width:44%"><span class="p">{{.Path}}</span></td>
      <td><span class="perm">{{.PermText}}</span></td>
      <td style="width:90px;text-align:right">
        {{if .Implemented}}<span class="tag done">已实现</span>{{else}}<span class="tag todo">二期</span>{{end}}
      </td>
    </tr>
    {{end}}
  </table>
</div>
{{end}}
</div>
<script>
function filter(){
  var q = document.getElementById('q').value.toLowerCase().trim();
  var cards = document.querySelectorAll('.card');
  for (var i=0;i<cards.length;i++){
    var card = cards[i], rows = card.querySelectorAll('tr'), shown = 0;
    for (var j=0;j<rows.length;j++){
      var hit = !q || rows[j].getAttribute('data-k').toLowerCase().indexOf(q) >= 0;
      rows[j].style.display = hit ? '' : 'none';
      if (hit) shown++;
    }
    var ctrlHit = !q || (card.getAttribute('data-ctrl')||'').toLowerCase().indexOf(q) >= 0;
    card.style.display = (shown || ctrlHit) ? '' : 'none';
  }
}
</script>
</body>
</html>`))
