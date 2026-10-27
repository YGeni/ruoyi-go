# AGENTS.md

本仓库正在做**渐进式迁移**：保留现有 Java（Spring Boot）后端，同时在 `server-go/` 下建设等价的 Go（Gin）后端。迁移完成前两套代码共存。

## 全局规则（任何 AI / 开发者都需遵守）

1. **禁止删除或修改 `ruoyi-admin` / `ruoyi-framework` / `ruoyi-system` / `ruoyi-common` / `ruoyi-quartz` / `ruoyi-generator` 中的现有代码**，只能读取作为迁移依据。
2. **禁止修改 `sql/` 下的表结构**。数据库 31 张表（20 张业务表 + 11 张 Quartz 表）是 Java 与 Go 两端的共同契约。
3. **Go 代码一律写在 `server-go/` 目录下**，拥有独立的 `go.mod`。
4. **禁止字符串拼接 SQL**。所有动态条件、数据权限片段必须使用 GORM 构造器 + 参数化。
5. **前端契约不可变**：统一返回体 `AjaxResult`、列表返回体 `TableDataInfo`、分页入参、401/403 语义必须与 `/tmp/ruoyi-ui-ref/ruoyi-ui` 的实际调用一致。
6. **不要一次性生成整个项目**。按 `docs/go-migration/` 下的阶段文档推进，每个阶段先列计划、经确认后再实现。
7. 与原 Java 实现存在行为差异时，必须在代码中留 `// DIFF(review): <说明>` 注释，**禁止静默偏离**。
8. 每个阶段结束必须自证：`go build ./...`、`go vet ./...` 零报错，附真实跑过的冒烟结果；跑不通要如实说明，不得谎称通过。

## 技术栈（已定，不要替换）

Gin + GORM + Casbin + golang-jwt/v5 + go-redis/v9 + robfig/cron/v3 + excelize + viper + zap + go-playground/validator + base64Captcha

遇到与 [gin-vue-admin](https://github.com/flipped-aurora/gin-vue-admin) 同构的问题，优先采用其成熟做法。

## 项目基线

| 项 | 值 |
| --- | --- |
| 项目根目录 | `/Users/mac/Desktop/若依/RuoYi-Vue-master` |
| 原版本 | RuoYi-Vue 3.9.2 / Spring Boot / Java 17 |
| 前端参考（不含本仓库） | `/tmp/ruoyi-ui-ref/ruoyi-ui` |
| 配置基线 | `ruoyi-admin/src/main/resources/application.yml`、`application-druid.yml` |
| 服务端口 / 库 | 8080、MySQL `ry-vue`、Redis db0 |

## 迁移文档索引

- `docs/go-migration/00-prepare.md` — 前置准备（一键脚本 `server-go/scripts/setup.sh`：拉前端、起 MySQL/Redis、初始化 31 张表）
- `docs/go-migration/01-review-prompt.md` — 第 0 期：交给 Codex 的**方案评审**提示词
- `docs/go-migration/02-phase1-prompt.md` — 第 1 期：骨架 + 认证授权落地提示词
- `docs/go-migration/03-module-prompt-template.md` — 后续模块通用提示词模板

## 待迁移机制清单（对照 Java 原实现）

| 机制 | 原实现位置 | 状态 |
| --- | --- | --- |
| 方法级鉴权 `@PreAuthorize("@ss.hasPermi(...)")` | 约 116 处 Controller | 待办 |
| 数据权限 `@DataScope` | `SysUserServiceImpl`(3) / `SysRoleServiceImpl` / `SysDeptServiceImpl` | 待办 |
| 定时任务 Quartz `invoke_target` 反射 | `ruoyi-quartz/` | 待办 |
| 操作日志 `@Log` + AOP | `framework/aspectj/LogAspect.java` | 待办 |
| Excel 导入导出 | `ruoyi-common/utils/poi/ExcelUtil.java` | 待办 |
| Velocity 代码生成器 | `ruoyi-generator/src/main/resources/vm/` | 待办 |
| XSS / Referer / Repeatable 过滤器 | `ruoyi-common/filter/` | 待办 |
| 动态数据源 `@DataSource` | slave 默认关闭，优先级低 | 待办 |
