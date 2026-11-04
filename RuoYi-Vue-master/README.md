# 天才管理系统

后端 **Go + Gin**，前端 **Vue 3 + Vite + Element Plus**。

本项目是把原本的 Spring Boot 后端（RuoYi-Vue）整体用 Go 重构后的结果：
**Java 模块已删除**，后端只剩 `server-go`。需要对照原实现时，从 git 历史导出到临时目录即可
（在**仓库根目录**执行，不会改动当前工作区；历史里的 Java 模块带 `RuoYi-Vue-master/` 前缀）：

```bash
mkdir -p /tmp/ruoyi-java
git archive 'fe08c90^' RuoYi-Vue-master | tar -x -C /tmp/ruoyi-java
```

导出结果在 `/tmp/ruoyi-java/RuoYi-Vue-master/`，含 `ruoyi-admin`、`ruoyi-common`、
`ruoyi-framework`、`ruoyi-generator`、`ruoyi-quartz`、`ruoyi-system` 六个模块。

## 项目特点

将若依原 Spring Boot 后端整体重构为 **Go + Gin + GORM + Redis**，编译生成独立单二进制，
消除对 JVM、Tomcat 的依赖，具备启动迅速、内存占用低、部署简单的特点。

- **鉴权链路**：基于 golang-jwt 签发 Token，登录态、验证码、字典与参数统一缓存到 Redis
- **行级数据权限**：通过 GORM `Scopes` 参数化拼装，对齐原 Java 端 `@DataScope` 的五种范围
- **数据一致性**：多表写入（通知、部门、代码生成等）使用 GORM 事务保证
- **数据库**：当前为 MySQL 8；GORM 的 Dialector 可替换，但尚未适配 Oracle 等方言

可作为 MES 等系统的配套信息化底座，适合厂区多节点轻量服务部署；
工单、台账等具体业务模块需在此基础上自行扩展。

## 开源许可与衍生说明

本项目基于 [RuoYi-Vue](https://gitee.com/y_project/RuoYi-Vue) 3.9.2 衍生开发，
原项目版权归 RuoYi（2018），采用 **MIT** 许可证，完整条款见 [LICENSE](./LICENSE)，
本仓库已保留原版权声明，并同样以 MIT 许可证开源。

相对原项目所做的修改：

- 后端由 Java 17 / Spring Boot 整体重构为 **Go 1.25 / Gin + GORM + go-redis**；
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

## 对比原 Spring Boot 版

### 运行时与部署

| 维度 | 原 Spring Boot 版 | Go 版 |
| --- | --- | --- |
| 运行时 | JDK + 内嵌 Tomcat | 无外部运行时 |
| 产物 | 86 MB fat jar（另需 JRE） | 46 MB 单二进制（含 Go 运行时与 embed 的 Swagger UI） |
| 配置 | `application.yml` + `application-druid.yml` | 单个 `config.yaml` |
| 启动过程 | Spring 上下文扫描、Bean 初始化、JVM 预热 | 初始化 DB/Redis 连接池后直接监听 |
| 部署 | 装 JDK、挑基础镜像、调 `Xmx`/GC | 拷一个文件 + 起 MySQL/Redis |
| 并发模型 | 线程池，每请求占一个线程 | goroutine，栈按需增长 |

没有 JVM 这一层，也就没有堆大小、元空间、GC 调优这些运维负担。组件对应关系：
Spring Security / MyBatis XML / Druid / Quartz / POI / OSHI / logback →
golang-jwt + 自建中间件 / GORM / `database/sql` 原生池 / 自研调度器（兼容 6 段 cron）/
excelize / gopsutil / zap。

### 本机实测

同一台机器、同一套 MySQL 8 + Redis 7 容器、同一个 `ry-vue` 库，各自冷启动一次，
以首个 `/captchaImage` 返回 200 为「可用」：

| 指标 | 原 Spring Boot 4.1.0（JDK 24） | Go 1.25 / Gin |
| --- | --- | --- |
| 冷启动到可用 | 16.57 s | 1.73 s |
| 空闲常驻内存（RSS） | 453.7 MB | 31.1 MB |
| 30 次请求后 RSS | 454.7 MB | 40.0 MB |
| 产物体积 | 86.1 MB（不含 JRE） | 46.2 MB（含运行时） |

> 数据说明：单次冷启动、非多轮平均；JVM 使用默认参数（项目 `pom.xml` 声明 Java 17，
> 实测环境为 JDK 24），加 `-Xmx` 或换 GC 会改变结果；产物体积两者口径不同，不可直接等价。
> 原 Java 版由 git 历史 `fe08c90^` 还原后重新构建，本机启动时把 logback 写死的
> `/home/ruoyi/logs` 改为临时目录。数字随硬件、负载与配置变化，仅供参考。

### 并发压测

两版串行运行、同一套容器、同一接口 `/captchaImage`，并发梯度 10 / 50 / 100，
每档预热 3 s 后压测 12 s，压测期间每 200 ms 采样一次进程 RSS。三档均 **0 失败**。

| 并发 | Go QPS | Java QPS | Go p95 / p99 | Java p95 / p99 |
| --- | --- | --- | --- | --- |
| 10 | 1297.7 | 450.8 | 15.5 / 26.3 ms | 30.0 / 37.1 ms |
| 50 | 1436.9 | 490.9 | 99.9 / 151.7 ms | 231.7 / 316.7 ms |
| 100 | 1514.3 | 638.6 | 211.9 / 318.3 ms | 371.7 / 482.0 ms |

| 并发 | Go RSS（起始 → 结束） | Java RSS（起始 → 结束） |
| --- | --- | --- |
| 10 | 75.4 → 84.5 MB | 507.3 → 571.6 MB |
| 50 | 109.6 → 121.7 MB | 571.8 → 575.6 MB |
| 100 | 141.6 → 153.2 MB | 579.9 → 573.2 MB |

> **口径提醒**：上表的内存是**有负载**口径。空闲刚启动时 Go 约 31 MB、Java 约 454 MB（≈14.6×），
> 但 100 并发稳态下为 153 MB vs 573 MB（≈3.7×）——Go 内存随并发线性增长，Java 是预分配堆、
> 几乎不随负载变化。引用时务必说明口径，不要把空闲值当成压测结论。
>
> 边界：压测客户端为 Go 编写且与服务端同机争 CPU，绝对值偏低；`/captchaImage` 两版实现不同
> （kaptcha vs base64Captcha）且都写 Redis，测的是「框架 + 运行时 + Redis」综合表现，
> 不是纯 Web 框架基准，Redis 很可能是共同瓶颈；JVM 使用默认参数（项目声明 Java 17，实测 JDK 24），
> Go 以本机 Go 1.26.5 编译（`go.mod` 声明 1.25）；每档单次运行、未做多轮平均，
> p99 受本机调度波动影响；100 并发下两版均 0 失败，未测到极限。

### 平滑迁移

- 已实现接口的路径与返回体保持兼容，前端无需改动，数据库表结构不变，可直接共用同一份库；
  少数接口属二期延后，详见[迁移状态](#迁移状态)
- OpenAPI 每个端点带 `x-ruoyi-implemented` 标记，`/swagger-ui/progress.html` 可查已实现 / 二期延后
- 与 Java 实现有意不同的地方在 Go 代码里以 `// DIFF(review)` 标注，权限串由
  `perm_diff_test.go` 与 Java 控制器源码做双向 diff 守护

### 尚未对齐（如实清单）

本项目不是 100% 等价替换。下面这些是**当前确实没做或只做了一半**的地方，
页面上看得到入口，但行为与原 Java 版不完全一致：

| 位置 | 现状 |
| --- | --- |
| 日志管理 → 操作日志 | 只有查询/删除/导出，**没有写入路径**：Go 侧未提供 `@Log` 对应的操作日志中间件，页面数据不会新增 |
| 各列表的导出 | 导出的是表头 + 裸数据行的 xlsx，**没有**列宽、字典回写、合计行、65536 行拆表；导入按固定列序，不含 601 错误聚合 |
| 缓存监控 | 「防重提交」「限流处理」两类缓存没有写入方（无对应中间件），页面显示为空，备注已标注 |
| 代码生成 | 只支持 `crud`；`sub` / `tree` 按 crud 渲染，Vue 模板未移植，生成物需按 `routes.txt` 的指引手工接线 |
| 定时任务 | 只支持内置注册表里的 `ryTask.*`；无 Spring 反射，cron 不支持 `L` / `W` / `#` / 年 字段 |
| 数据监控 | 自建对等页，无 druid 的 SQL 防火墙、慢 SQL 阈值、SELECT 读取行数统计 |
| 参数设置 → 验证码开关 | 已与登录校验共用同一个配置源，关闭后前后端都不再要求验证码 |
| 二期延后接口 | 5 条 `/test/user/*` 是 Java 演示控制器，未注册；Swagger UI 的 Try it out 点到它们会 404，请用进度页区分 |
| `config.yaml` | `slave` 多数据源未实现；`xss` / `referer` 中间件未提供，配置项仅为对齐占位且 `enabled: false`；`initialSize` / `maxWait` 在 `database/sql` 下无对应语义 |
| 前端「表单构建」 | 原版即纯前端页面，本来就没有后端接口 |

其余有意差异在 Go 代码里以 `// DIFF(review)` 标注。另外：

- 数据库目前只适配 MySQL
- 本项目是个人衍生项目，没有 RuoYi 官方的持续维护与插件生态

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
| `sql/ry_20260417.sql` 的 `sys_user` | `admin` / `admin123`、`ry` 两个种子账号 | 密码哈希是 RuoYi 公开默认值，上线前删除 `ry` 并修改 `admin` 密码 |
| `sys.user.initPassword` | `123456` | 新建/重置用户的初始密码，建议一并调整并强制首次修改 |
| 前端 `src/utils/jsencrypt.js` | RuoYi 原版公开的 RSA 公钥/私钥对 | 私钥随前端打包，「记住密码」的本地加密不具备机密性；生产应改为后端下发公钥或去掉本地加解密 |

同时建议：MySQL / Redis 不要用默认端口直接暴露到公网；`admin` 账号的初始密码 `admin123` 首次登录后立即修改。
前端登录页的默认账号密码已改为**仅开发模式**（`import.meta.env.DEV`）预填，不会进生产构建。

## 接口文档

| 地址 | 说明 |
| --- | --- |
| `/v3/api-docs` | OpenAPI 3.0 JSON，可导入 Postman / Apifox |
| `/swagger-ui/index.html` | 真 Swagger UI（静态资源已打进二进制，离线可用），支持 Try it out |
| `/swagger-ui/progress.html` | 自绘清单，带「已实现 / 二期延后」标记 |

以上地址均校验登录态（`authForIframe`）：**裸开会 401**。最省事的方式是登录后从后台
「系统工具 → 系统接口」菜单进入（前端 iframe 自动把 Token 拼到 URL）；直接访问或导入
Postman / Apifox 时需在 URL 后加 `?token=<登录 Token>`，Token 可从浏览器
Local Storage 的 `Admin-Token` 取得。

请求体 schema 采用「注册表 + 反射」：注册表只声明每条路由对应哪个 Go 结构体，
字段清单、类型与必填由结构体反射得到，字段说明取自 **MySQL 列注释**；
因此改结构体会自动反映到文档，但新增路由仍需在注册表登记一条，不是零维护。

## 迁移状态

原 Java 端共 **147 个端点**：已实现 **142**，二期延后 **5**（以进度页为准）。
其中 `getInfo` / `getRouters` 在 Java 注解里没有前导斜杠，Go 侧注册的是 `/getInfo`、
`/getRouters`；生成文档时会规范化路径，避免这两条已实现的端点被误标为二期。
剩余 5 条是 Java 的 `/test/user/*` 演示控制器，Go 侧未注册。

路由与权限串登记在 `internal/router/perm_table.go`，由 `perm_diff_test.go`
双向对账（基准快照 `internal/router/testdata/java_endpoints.json`），
漏绑或抄错权限串会直接测试失败。

## 说明

- Go 代码里的 `DIFF(review)` 注释记录的是"与原 Java 实现有意不同的地方"
  （例如 SQL 监控改用 gorm 回调打点、服务监控那块是 Go 运行时而非 JVM），
  不是待办。
- 服务监控的 SQL 统计在 `pkg/sqlstats`，按归一化后的 SQL 模板聚合；
  读取行数、SQL 防火墙这类 druid 专有能力没有对应实现。
