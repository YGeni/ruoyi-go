// Package assets 打进二进制的静态资源。
//
// 存在的理由：Swagger UI 的 JS/CSS 如果走 CDN，离线或内网环境就是一片白屏；
// 打进二进制后不依赖任何外部网络，也没有运行时文件目录的部署要求。
package assets

import "embed"

// SwaggerUI swagger-ui-dist 的静态文件目录。
//
// 内容取自 unpkg 的 swagger-ui-dist@5.17.14（MIT 协议）：
//
//	swagger-ui.css                 样式
//	swagger-ui-bundle.js           主包（含 React 运行时与 swagger-client）
//	swagger-ui-standalone-preset.js Standalone 布局预设（顶栏、Authorize 按钮）
//	favicon-32x32.png / favicon-16x16.png
//
// 合计约 1.8MB，会体现在二进制体积上。
//
//go:embed swagger-ui
var SwaggerUI embed.FS
