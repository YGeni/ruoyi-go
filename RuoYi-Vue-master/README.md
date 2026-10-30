# 天才管理系统

后端 **Go + Gin**，前端 **Vue 3 + Vite + Element Plus**。

本项目是把原本的 Spring Boot 后端（RuoYi-Vue）整体用 Go 重写后的结果：
**Java 模块已删除**，后端只剩 `server-go`。需要对照原实现时，从 git 历史取回即可：

```bash
git checkout fe08c90^ -- ruoyi-admin ruoyi-generator ruoyi-quartz
```

## 开源许可与衍生说明

本项目基于 [RuoYi-Vue](https://gitee.com/y_project/RuoYi-Vue) 3.9.2 衍生开发，
原项目版权归 RuoYi（2018），采用 **MIT** 许可证，完整条款见 [LICENSE](./LICENSE)，
本仓库已保留原版权声明，并同样以 MIT 许可证开源。

相对原项目所做的修改：

- 后端由 Java 17 / Spring Boot 全量重写为 **Go 1.25 / Gin + GORM + go-redis**；
  原 Java 模块（`ruoyi-admin`、`ruoyi-generator`、`ruoyi-quartz` 等）已从仓库移除，可从 git 历史取回
- 前端（Vue 3 + Vite + Element Plus）基本沿用原实现，仅调整少量接口兼容与文案
- 数据库表结构（31 张表）保持不变，可与原 Java 端共用同一份库
- 与原 Java 实现有意不同的地方，在 Go 代码中以 `// DIFF(review)` 注释标注

项目名称「天才管理系统」与原项目名称无关，本项目不宣称官方身份，也不代表 RuoYi 官方立场。

## 技术栈

| 层 | 选型 |
| --- | --- |
| 后端 | Go 1.25 / Gin / GORM / go-redis / Viper / zap |
| 认证 | JWT（golang-jwt），Token 存 Redis |
| 系统指标 | gopsutil（服务监控） |
| 导出 | excelize（Excel） |
| 前端 | Vue 3 + Vite + Element Plus + Pinia（见 `../RuoYi-Vue3-master`） |
| 存储 | MySQL 8 + Redis 7（docker compose 起） |

## 目录

```
server-go/
  cmd/server/        入口（main）
  config/            配置读取 + config.yaml
  internal/
    router/          路由注册、权限表、OpenAPI 生成、Swagger UI
    handler/         HTTP 层
    service/         业务层
    repository/      数据访问层
    model/           entity / query
    scheduler/       定时任务（quartz 风格 6 段表达式）
  pkg/               可复用包（db / redisx / jwtx / page / sqlstats / osx ...）
  assets/            swagger-ui 静态资源（go:embed 打进二进制）
sql/                 建表 + 种子数据（容器首次启动自动导入）
docs/go-migration/   迁移过程文档与自动生成的权限 diff 报告
```

## 跑起来

```bash
# 1) MySQL + Redis
docker compose -f server-go/docker-compose.yml up -d

# 2) 后端（:8080）
cd server-go
go run ./cmd/server -c config/config.yaml     # 或 make run
air                                           # 或 make dev（改代码自动重启）

# 3) 前端（:80，代理 /dev-api -> localhost:8080）
cd ../../RuoYi-Vue3-master
npm install && npm run dev
```

打开 http://localhost ，账号 `admin` / `admin123`。

常用：`make build`（编译到 `bin/server`）、`make vet`、`make test`。

## 上线前必改

`server-go/config/config.yaml` 里的这些值是**开发默认值，不要直接用于生产**：

| 配置 | 默认值 | 说明 |
| --- | --- | --- |
| `token.secret` | `abcdefghijklmnopqrstuvwxyz` | JWT 签名密钥，泄露即可伪造任意用户登录态 |
| `datasource.master.password` | `password` | 数据库密码，需与 `docker-compose.yml` 中 MySQL 的密码一致 |
| `redis.password` | 空 | 若 Redis 启用了密码认证需同步填写 |

同时建议：MySQL / Redis 不要用默认端口直接暴露到公网；`admin` 账号的初始密码 `admin123` 首次登录后立即修改。

## 接口文档

| 地址 | 说明 |
| --- | --- |
| `/v3/api-docs` | OpenAPI 3.0 JSON，可导入 Postman / Apifox |
| `/swagger-ui/index.html` | 真 Swagger UI（静态资源已打进二进制，离线可用），支持 Try it out |
| `/swagger-ui/progress.html` | 自绘清单，带「已实现 / 二期延后」标记 |

请求体字段说明取自 **MySQL 列注释**、由 Go 结构体反射生成，不是手写维护的。

## 迁移状态

原 Java 端共 **147 个端点**：已实现 130，二期延后 17（Excel 导入导出等）。

路由与权限串登记在 `internal/router/perm_table.go`，由 `perm_diff_test.go`
双向对账（基准快照 `internal/router/testdata/java_endpoints.json`），
漏绑或抄错权限串会直接测试失败。

## 说明

- Go 代码里的 `DIFF(review)` 注释记录的是"与原 Java 实现有意不同的地方"
  （例如 SQL 监控改用 gorm 回调打点、服务监控那块是 Go 运行时而非 JVM），
  不是待办。
- 服务监控的 SQL 统计在 `pkg/sqlstats`，按归一化后的 SQL 模板聚合；
  读取行数、SQL 防火墙这类 druid 专有能力没有对应实现。
