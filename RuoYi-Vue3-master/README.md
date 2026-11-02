<p align="center">
  <img alt="logo" src="./src/assets/logo/logo.png" width="120">
</p>
<h1 align="center" style="margin: 30px 0 30px; font-weight: bold;">天才管理系统 v4.0.0</h1>
<h4 align="center">前端 Vue 3 + Vite + Element Plus，配套 Go（Gin）后端</h4>
<p align="center">
  <a href="https://github.com/YGeni/ruoyi-go">GitHub 仓库</a> ·
  <a href="../RuoYi-Vue-master/README.md">后端说明</a> ·
  <a href="./LICENSE">MIT</a>
</p>

## 平台简介

- 本目录是前端工程，技术栈 [Vue 3](https://v3.cn.vuejs.org) + [Element Plus](https://element-plus.org/zh-CN) + [Vite](https://cn.vitejs.dev) + Pinia，沿用 RuoYi-Vue3 的结构，未做结构性改动。
- 配套后端是同级的 [RuoYi-Vue-master/server-go](../RuoYi-Vue-master/server-go)：原 Java / Spring Boot 后端已用 **Go + Gin + GORM** 整体重构。已实现接口的路径与返回体保持兼容，前端无需改动即可对接；少量接口属二期延后，以[迁移进度](../RuoYi-Vue-master/README.md#迁移状态)为准。
- 本项目基于 [RuoYi-Vue](https://gitee.com/y_project/RuoYi-Vue) 3.9.2 衍生开发，原项目为 MIT 许可证（Copyright (c) 2018 RuoYi），版权声明见 [LICENSE](./LICENSE)，本仓库沿用 MIT。项目名称「天才管理系统」与原项目无关，不宣称官方身份，也不代表 RuoYi 官方立场。

## 本地运行

仓库根目录同时包含前端 `RuoYi-Vue3-master/` 与后端 `RuoYi-Vue-master/`。先按[后端 README 的「跑起来」](../RuoYi-Vue-master/README.md#跑起来)起好 MySQL / Redis 和 Go 服务（`:8080`），再启动前端：

```bash
git clone https://github.com/YGeni/ruoyi-go.git
cd ruoyi-go/RuoYi-Vue3-master

npm install
npm run dev            # http://localhost（Vite 端口 80，/dev-api 代理到 :8080）

npm run build:stage      # 构建测试环境
npm run build:prod       # 构建生产环境
```

打开 http://localhost ，账号 `admin` / `admin123`。这是开发默认账号，上线前必改，见[后端 README 的「上线前必改」](../RuoYi-Vue-master/README.md#上线前必改)。

## 内置功能

1. 用户管理：配置系统操作用户。
2. 部门管理：配置组织机构（公司、部门、小组），树结构展现并支持数据权限。
3. 岗位管理：配置用户所属职务。
4. 菜单管理：配置系统菜单、操作权限与按钮权限标识。
5. 角色管理：角色菜单权限分配、按机构划分数据范围权限。
6. 字典管理：维护系统中较固定的数据字典。
7. 参数管理：维护系统动态配置参数。
8. 通知公告：通知公告的发布与维护。
9. 操作日志：正常操作与异常信息的记录和查询。
10. 登录日志：登录记录查询，包含登录异常。
11. 在线用户：当前活跃用户状态监控。
12. 定时任务：在线添加、修改、删除任务调度，含执行结果日志；由 Go 自研调度器实现，兼容 Quartz 风格 6 段 cron。
13. 代码生成：生成 Go 后端分层代码（entity / query / repository / service / handler / 路由条目）与菜单 SQL，支持 CRUD 下载。
14. 系统接口：内置 Swagger UI 与 OpenAPI 3.0 文档，字段说明由「注册表 + Go 结构体反射」生成，非逐端点手写。
15. 服务监控：CPU、内存、磁盘与 Go 运行时状态（gopsutil 采集，非 JVM 堆栈指标）。
16. 缓存监控：Redis 缓存信息与命令统计。
17. 在线构建器：拖动表单元素生成 HTML 代码。
18. 连接池监视：数据库连接池状态与归一化 SQL 统计；Druid 的 SQL 防火墙、读取行数等专有能力未实现。

## 接口文档

后端提供 `/v3/api-docs`、`/swagger-ui/index.html`、`/swagger-ui/progress.html`，均需登录态。
推荐从后台「系统工具 → 系统接口」菜单进入（iframe 自动带 Token）；直接访问需在 URL 后拼 `?token=<登录 Token>`。
