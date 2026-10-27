# AGENTS.md

本仓库正在做**渐进式迁移**：保留现有 Java（Spring Boot）后端，同时在 `server-go/` 下建设等价的 Go（Gin）后端。迁移完成前两套代码共存。

## 全局规则（任何 AI / 开发者都需遵守）

1. **禁止删除或修改 `ruoyi-admin` / `ruoyi-framework` / `ruoyi-system` / `ruoyi-common` / `ruoyi-quartz` / `ruoyi-generator` 中的现有代码**，只能读取作为迁移依据。
2. **禁止修改 `sql/` 下的表结构**。数据库 31 张表（20 张业务表 + 11 张 Quartz 表）是 Java 与 Go 两端的共同契约。
3. **Go 代码一律写在 `server-go/` 目录下**，拥有独立的 `go.mod`。
4. **禁止字符串拼接 SQL**。所有动态条件、数据权限片段必须使用 GORM 构造器 + 参数化。
5. **前端契约不可变**：统一返回体 `AjaxResult`、列表返回体 `TableDataInfo`、分页入参、401/403 语义必须与 `/Users/mac/Desktop/若依/RuoYi-Vue3-master`（**Vue3 / Element Plus / Vite**）的实际调用一致。
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
| 前端参考（相邻目录） | `/Users/mac/Desktop/若依/RuoYi-Vue3-master`（**Vue3**，不是 Vue2） |
| 服务端口 | 8080（`vite.config.js:5` 的 proxy target） |
| 配置基线 | `ruoyi-admin/src/main/resources/application.yml`、`application-druid.yml` |
| 服务端口 / 库 | 8080、MySQL `ry-vue`、Redis db0 |

## 前端契约锚点（一律以 Vue3 前端源码为准）

- `src/utils/request.js:32` — 请求头恒定带 `Authorization: Bearer <token>`，**Go 侧必须剥离 `Bearer ` 前缀**再验签（原实现见 `ruoyi-framework/.../web/service/TokenService.java:219-227`）
- `src/utils/request.js:78-109` — code 语义：200 成功 / 401 会话失效重登 / 500 错误 / 601 警告，其余非 200 一律 reject
- `src/store/modules/permission.js:35-97` — `getRouters` 返回树的解析方式：`component` 取 `Layout` / `ParentView` / `InnerLink` 或 views 相对路径；叶子节点会被 `delete children` 与 `delete redirect`
- `src/views/login.vue:143-150` — 验证码接口返回 `{ captchaEnabled, img, uuid }`，`img` 为**不含** `data:image/gif;base64,` 前缀的裸 base64，前端自行拼接
- `vite.config.js:5,48-54` — dev 代理目标 `http://localhost:8080`，并去除 `/dev-api` 前缀

## 迁移文档索引

- `docs/go-migration/00-prepare.md` — 前置准备（一键脚本 `server-go/scripts/setup.sh`：拉前端、起 MySQL/Redis、初始化 31 张表）
- `docs/go-migration/01-review-prompt.md` — 第 0 期：交给 Codex 的**方案评审**提示词
- `docs/go-migration/02-phase1-prompt.md` — 第 1 期：骨架 + 认证授权落地提示词
- `docs/go-migration/03-module-prompt-template.md` — 后续模块通用提示词模板
- `AGENTS.md` §7–§12 — **方案评审结论（2026-10-28 定稿）**：目录结构 / go.mod / 三期计划 / 8 个 Java 机制的 Go 替代 / 前端契约清单 / 三大风险。落地实现前先读这一节。

## 待迁移机制清单（摘要，详细方案见 §10）

| 机制 | 原实现位置（实测） | Go 替代 | 状态 |
| --- | --- | --- | --- |
| 方法级鉴权 `@PreAuthorize("@ss.hasPermi(...)")` | `framework/web/service/PermissionService.java:18`（`@Service("ss")`），**115 处** `@ss.hasPermi` / 17 文件 | 自写 `Permi` 中间件 + Redis 权限集（Casbin 可选） | 待办 |
| 数据权限 `@DataScope` | `framework/aspectj/DataScopeAspect.java`；落点 `SysUserServiceImpl:76/89/102`、`SysRoleServiceImpl:54`、`SysDeptServiceImpl:45` | GORM `Scopes` + 参数化（禁字符串拼 SQL） | 待办 |
| 定时任务 `invoke_target` 反射 | `ruoyi-quartz/util/JobInvokeUtil.java:23-141`；`config/ScheduleConfig.java` **整文件被注释**（实为 RAM JobStore） | robfig/cron/v3 `WithSeconds()` + 任务注册表 | 待办 |
| 操作日志 `@Log` + AOP | `framework/aspectj/LogAspect.java` | 路由元数据 + `OpLog` 中间件 + 异步 worker 池 | 待办 |
| Excel 导入导出 | `ruoyi-common/utils/poi/ExcelUtil.java`（71KB）+ `@Excel`（30+ 属性） | excelize + struct tag + 反射 | 待办 |
| Velocity 代码生成器 | `ruoyi-generator/src/main/resources/vm/`（21 个模板） | `text/template`，定界符改 `[[ ]]` | 待办 |
| XSS / Referer / Repeatable 过滤器 | `ruoyi-common/filter/*.java`，注册于 `framework/config/FilterConfig.java:35-78` | 三个 Gin 中间件 | 待办 |
| 动态数据源 `@DataSource` | `framework/aspectj/DataSourceAspect.java`；`application-druid.yml` slave `enabled: false` | 单库优先，`pkg/db.DB(ctx)` 预留切换 | 待办 |

---

# 方案评审结论（2026-10-28 定稿，实测源码得出，非外部记忆）

> 结论摘要：后端真实入口是 ruoyi-admin 下 **18 个带 `@RequestMapping` 前缀的 Controller** + 无前缀的 `SysLoginController`/`CaptchaController`；`GenController` 在 `ruoyi-generator/.../controller/GenController.java`，不属于 ruoyi-admin。鉴权 Bean 是 `@Service("ss")` 的 `PermissionService`（**不是** `SysPermissionService`），`*:*:*` 直通在 `PermissionService.java:157`，`admin` 直通在 `hasRole` L102。数据权限是 AOP 拼串后以 `${params.dataScope}` 注入（`SysUserMapper.xml:86/103/121`），Go 侧必须改为 GORM `Scopes` 参数化。Quartz 的 `ScheduleConfig.java` 整文件被注释 → 实际 RAM JobStore，`QRTZ_*` 11 表无数据，换 robfig/cron 无历史包袱，但 cron 为 **6 段含 `?`**（`0/10 * * * * ?`），须 `WithSeconds()` 并把 `?` 归一为 `*`。策略：①鉴权用自写中间件 + Redis 权限集（库内无 `casbin_rule`，Casbin 只作可选）；②按 admin→handler/router、framework→middleware/pkg、system→service/repository/model、common→pkg、quartz→job、generator→generator+templates 映射；③端口锁死 8080，返回体/分页/导出头严格对齐前端；④三期推进：认证+系统管理 → 监控+任务+Excel/文件 → 生成器+双跑切换。三大风险：数据权限语义漂移、Excel/生成器反射元数据对等、时间/`char(1)` 字段与前端隐性契约。

## 7. `server-go/` 目录结构与分层

现状：`server-go/` 目前只有 `scripts/setup.sh` + `docker-compose.yml`，从零搭建。

```
server-go/
├── cmd/server/main.go                 # 启动：viper → zap → gorm → redis → cron → gin
├── config/
│   ├── config.yaml                    # 对齐 application.yml + application-druid.yml
│   ├── config.go                      # Server/Jwt/Redis/MySQL/Token/User/Captcha/Xss/Referer
│   └── constant.go                    # CacheConstants / HttpStatus / UserConstants 移植
├── internal/
│   ├── handler/                       # ← ruoyi-admin 全部 Controller
│   │   ├── system/{user,role,dept,menu,post,dict_type,dict_data,config,notice,profile,login}.go
│   │   ├── monitor/{operlog,logininfor,online,cache,server}.go
│   │   ├── monitor/job.go, job_log.go # ← ruoyi-quartz/.../SysJobController.java
│   │   ├── tool/gen.go                # ← ruoyi-generator/.../GenController.java
│   │   └── common/{captcha,upload,download}.go
│   ├── service/                       # ← ruoyi-system 13 Service + quartz + generator
│   ├── repository/                    # ← ruoyi-system 16 Mapper（GORM）
│   ├── model/{entity,vo,dto,query}/   # ← entity + RouterVo/MetaVo + PageDomain/TableSupport
│   ├── middleware/                    # ← ruoyi-framework：jwt/permi/datascope/oplog/
│   │                                 #   xss/referer/repeatable/cors/ratelimit
│   ├── router/
│   │   ├── router.go                  # 路由注册
│   │   └── perm_table.go              # 路由 → 权限串映射（替代 115 处注解）
│   └── job/                           # ← ruoyi-quartz：cron manager / task registry
├── pkg/                               # ← ruoyi-common 全部工具类
│   ├── result/                        # AjaxResult + TableDataInfo（200/401/403/500/601）
│   ├── page/                          # pageNum/pageSize/orderByColumn/isAsc/reasonable
│   ├── jwtx/ redisx/ securityx/       # TokenService / RedisCache / SecurityUtils(BCrypt)
│   ├── excel/ captcha/ dict/ ip/ file/
│   ├── logger/ async/                 # zap / AsyncManager 异步池
│   └── generator/                     # VelocityUtils + GenUtils 移植
├── templates/                         # 21 个 vm 模板的去 Velocity 版
├── scripts/setup.sh  docker-compose.yml（已存在）
├── go.mod  Makefile
```

**六模块职责对应**

| 原模块 | Go 落点 | 依据 |
| --- | --- | --- |
| ruoyi-admin | `handler/` + `router/` | `ruoyi-admin/src/main/java/com/ruoyi/web/controller/**`，实测 18 个 `@RequestMapping` 前缀 + `SysLoginController.java:56/72/101`（无前缀） |
| ruoyi-framework | `middleware/` + `pkg/jwtx|redisx|securityx|async` + `config/` | `SecurityConfig.java:96-116`、`TokenService.java`、`FilterConfig.java:35-78`、`ThreadPoolConfig.java:20-61` |
| ruoyi-system | `service/` + `repository/` + `model/` | 13 Service / 16 Mapper 在 `ruoyi-system/src/main/java/com/ruoyi/system/{service,mapper}`；`SysUserOnlineServiceImpl` 无表，扫 `login_tokens:*` |
| ruoyi-common | `pkg/` | `AjaxResult.java`、`ExcelUtil.java`、`DictUtils.java`、`FileUtils.java:224-226` |
| ruoyi-quartz | `internal/job/` + `service/job*.go` | `SysJobServiceImpl.java:35-46` 启动时重建；`JobInvokeUtil.java:23-141` |
| ruoyi-generator | `pkg/generator/` + `templates/` + `service/gen.go` | `VelocityUtils.java:145-193`、`ruoyi-generator/src/main/resources/vm/**`（21 个） |

## 8. go.mod 依赖清单

```go
module github.com/YGeni/ruoyi-go/server-go
go 1.25  // 本机实测 go1.26.5；不得写 1.22，否则语言特性被锁死（1.23+ 的 range-over-func 等不可用）

require (
  github.com/gin-gonic/gin                  v1.10.x  // HTTP 框架，替代 Spring MVC
  gorm.io/gorm                              v1.25.x  // ORM，替代 MyBatis（16 Mapper → Repository）
  gorm.io/driver/mysql                      v1.5.x   // MySQL 驱动，替代 Druid（application-druid.yml master）
  github.com/casbin/casbin/v2                v2.100.x // 可选：仅在引入 casbin_rule 策略表时使用（见 §10.a）
  github.com/golang-jwt/jwt/v5               v5.2.x   // JWT HS512，对齐 TokenService.java:183
  github.com/redis/go-redis/v9               v9.7.x   // login_tokens:/captcha_codes:/sys_dict:/pwd_err_cnt:
  github.com/robfig/cron/v3                  v3.0.x   // 替代 Quartz（sys_job 表驱动）
  github.com/xuri/excelize/v2                v2.9.x   // 替代 POI ExcelUtil
  github.com/spf13/viper                     v1.19.x  // 配置，对齐 application.yml + application-druid.yml
  go.uber.org/zap                            v1.27.x  // 日志
  github.com/go-playground/validator/v10     v10.22.x // 替代 @Validated（如 SysNoticeController.java:68）
  github.com/mojocn/base64Captcha            v1.3.x   // 替代 Kaptcha（CaptchaConfig.java:18-82，math/char）

  // 必要补充
  golang.org/x/crypto                        v0.26.x  // bcrypt，对齐 SecurityUtils.java:92-115
  github.com/google/uuid                     v1.6.x   // token / 导出文件名 UUID+"_"+sheetName
  gopkg.in/yaml.v3                           v3.0.x   // viper 配置解析
)
// 可选：gorm.io/plugin/dbresolver（多数据源，见 §10.h）；github.com/stretchr/testify（契约测试）
```

**不引入更多依赖的依据**：原工程无 MQ/ES/WebSocket；`sql/` 全库无视图、存储过程、触发器、函数（grep 零命中）。

## 9. 三期实施计划

### 一期：能登录、能管系统（前端 80% 页面可跑）

**实施顺序（必须拆两步，禁止一期一次性铺完）**

- **1a（首个里程碑）**：`config/` + `pkg/{result,page,jwtx,redisx,securityx,logger}` + `cmd/server/main.go`；`middleware/{jwt,permi,cors}`；`handler/system/login.go`、`handler/system/menu.go`（`getRouters`）、`handler/common/captcha.go`；`model/entity` 与 `repository` 仅需覆盖 user/role/menu/dept。**验收：前端能登录并看到菜单。**
- **1b**：`middleware/datascope` + 数据权限五分支单测；user/role/dept/post/dict/config/notice 完整 CRUD；`handler/common/{upload,download}` 与 `profile`。

**产出物**（1a + 1b 合计）：`config/` + `pkg/{result,page,jwtx,redisx,securityx,logger,async}` + `cmd/server/main.go`；`middleware/{jwt,permi,datascope,cors,repeatable}` + `router/perm_table.go`；`handler/system/*` + `handler/common/{captcha,upload,download}`；20 张表 entity + repository。
**验收标准**
- `POST /login` → `{code:200,msg,token}`（`SysLoginController.java:56-65`，字段名来自 `Constants.TOKEN`）
- `GET /captchaImage` → `{captchaEnabled,uuid,img}`，`img` 裸 base64（`CaptchaController.java:90-91`）
- `GET /getInfo` 必须含 **user / roles / permissions / pwdChrtype / isDefaultModifyPwd / isPasswordExpired** 六字段（`SysLoginController.java:86-93`）
- `GET /getRouters` → `{data:[RouterVo]}`，`component` 三值齐备（`UserConstants.java:58-64`）
- 5 种数据权限用 admin/自定义/本部门/本部门及以下/仅本人 账号逐一比对列表行数，与 Java 版一致
- 401 对齐 `AuthenticationEntryPointImpl.java:30-32`；403 对齐 `GlobalExceptionHandler.java:35-41`（"没有权限，请联系管理员授权"）
**可联调**：登录/登出/验证码、首页路由菜单、用户/角色/部门/岗位/菜单/字典/参数/通知 CRUD、个人中心与头像、在线用户、操作日志与登录日志查询。

### 二期：监控、调度、Excel、文件
**产出物**：`internal/job/`（cron + 注册表 + `sys_job_log`）、`handler/monitor/*`、`pkg/excel/`、`middleware/{oplog,xss,referer,ratelimit}`、`handler/common/*`。
**验收标准**
- 三条种子任务（`sql/ry_20260417.sql:600-602`：`ryTask.ryNoParams`、`ryParams('ry')`、`ryMultipleParams('ry', true, 2000L, 316.50D, 100)`）跑通且 `sys_job_log` 落库字段齐全
- 导出 xlsx 与 Java 版**逐格 diff**：列宽 `(width+0.72)*256`、`dateFormat`、`dictType` 回写、`readConverterExp` 反查、`isStatistics` 合计行
- 导出响应头：`Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet;charset=utf-8` + `download-filename` + `Access-Control-Expose-Headers: Content-Disposition,download-filename`（`FileUtils.java:224-226`；前端 `plugins/download.js:13-40` 只读 `download-filename`）
- `/common/upload` 返回 `url/fileName/newFileName/originalFilename`（`CommonController.java:84-89`）
**可联调**：定时任务页（CRUD/启停/执行一次/日志）、全部列表导出与导入、缓存监控、服务器监控、文件上传下载。

### 三期：代码生成器 + 双跑切换
**产出物**：`pkg/generator/` + `templates/`（21 个）+ `handler/tool/gen.go`（含 zip：`GenController.java:256-261` 的 `Content-Disposition: attachment; filename="ruoyi.zip"` 与 `Access-Control-Expose-Headers: Content-Disposition`）；双跑对比脚本 + 契约测试集 + 切流方案。
**验收标准**：`gen_table`/`gen_table_column`（`sql/ry_20260417.sql:668-723`）导入 → 预览 → 下载 zip；前端模板产物与 Java 版逐字节 diff（除定界符）；Java 与 Go 同库双跑 7 天，`sys_oper_log` 条数与 `cost_time` 可比。

## 10. 八个 Java 机制的 Go 替代方案

### a) `@PreAuthorize("@ss.hasPermi(...)")` → **自写中间件 + Redis 权限集**（Casbin 仅可选）
- 事实：Bean 是 `PermissionService`（`framework/web/service/PermissionService.java:18` `@Service("ss")`）；`SysPermissionService.java:58-88` 只负责装载权限集。直通 `permissions.contains("*:*:*")` 在 `PermissionService.java:157`（`Constants.java:76`）；`hasRole` 另有 `SUPER_ADMIN="admin"` 直通（`:102`）。实测 `@ss.hasPermi` **115 处 / 17 文件**（Role 15、User 12、Gen 12、Job 8、Config 7、Dept 7、DictType 7、Cache 7、Menu 6、Post 6、DictData 6、Notice 5、JobLog 5、Logininfor 5、Operlog 4、Online 2、Server 1）；`@PreAuthorize` 全量 116（.java）+ 6（vm）。
- 方案：①路由注册即绑定权限 `engine.POST("/system/user/list", Permi("system:user:list"), h.List)`，115 条集中在 `router/perm_table.go`；②中间件判定 `perms.Contains("*:*:*") || perms.Contains(perm)`，否则 `code:403` + "没有权限，请联系管理员授权"；③权限集随 Redis `login_tokens:<uuid>` 的 LoginUser 一起存（`CacheConstants.java:13`），改角色/菜单时复用 `TokenService.java:240-269` 扫 `login_tokens:*` 重刷；④支持 `hasAnyPermi`（分隔符 `,`，`Constants.java:86/91`）与 `hasRole/hasAnyRoles`。
- **不用 Casbin 作主方案的理由**：31 张表内**没有 `casbin_rule`**，需自建策略表或每次启动反向生成 policy；且决策只有"权限串集合包含"，无 RBAC 继承、无 ABAC/资源实例级判断。gin-vue-admin 用 Casbin 是因为它自带策略表，本仓库不具备。折中：封装 `authz.Enforcer` 接口，Casbin 作为可插拔实现备用。

### b) `@DataScope` → GORM `Scopes`，参数化
- 事实：注解 `common/annotation/DataScope.java:22-42`（`userAlias/deptAlias/userField="user_id"/deptField="dept_id"/permission`）；切面 `framework/aspectj/DataScopeAspect.java` L50 admin 直通不拼、L143 注入 `" AND (" + sqlString.substring(4) + ")"`（去首个 `" OR "`）。五分支（`Constants.java:182-202`）：`"1"` 清空（L92-94）、`"2"` `OR {alias}.dept_id IN (SELECT dept_id FROM sys_role_dept WHERE role_id in (...))`（L101/105）、`"3"` `OR d.dept_id = {user.deptId}`（L110）、`"4"` `OR d.dept_id IN (SELECT dept_id FROM sys_dept WHERE dept_id=? or find_in_set(?, ancestors))`（L114）、`"5"` `OR {userAlias}.user_id = {userId}`，无别名时 `OR d.dept_id = 0`（L120/125）；L132-135 **空条件兜底 `OR d.dept_id = 0`（不查任何数据）**。落点 `SysUserServiceImpl.java:76/89/102`、`SysRoleServiceImpl.java:54`、`SysDeptServiceImpl.java:45`；消费 `SysUserMapper.xml:86,103,121`、`SysRoleMapper.xml:55`、`SysDeptMapper.xml:46`。
- 方案：`func DataScope(ctx, u *LoginUser, aliasDept, aliasUser string) func(*gorm.DB) *gorm.DB`，用 `Or(...)` 链 + `?` 占位（原 Java 为 `${}` 拼串，此处一并对齐规则 4）；`find_in_set` 用 `FIND_IN_SET(?, ancestors)`；兜底 `d.dept_id = 0` 必须复刻；`permission` 来源由 `Permi` 中间件写入 `context`（等价 Java 的 request-scoped `PermissionContextHolder`，`PermissionService.java:38/70`）。5 个调用点逐个写单测。

### c) Quartz 表驱动 → robfig/cron/v3 + 注册表 + 反射参数
- 事实：`JobInvokeUtil.java:23-39` 取 bean/方法；参数切分正则 `split(",(?=([^\"']*[\"'][^\"']*[\"'])*[^\"']*$)")`；类型推断（引号→String、`true/false`→Boolean、`L`→Long、`D`→Double、其余→Integer，`:71-141`）。白名单仅 `com.ruoyi.quartz.task`，黑名单含 `rmi://`、`ldap`、`http(s)://`、`java.net.URL`、`org.springframework` 等（`Constants.java:143-172`）。`concurrent` 0允许/1禁止、`misfire_policy` 1立即/2执行一次/3放弃（`sql/ry_20260417.sql:589-590`）。**`ruoyi-quartz/config/ScheduleConfig.java` 整文件被注释 → 实际 RAM JobStore，`QRTZ_*` 11 表无数据**；`SysJobServiceImpl.java:35-46` 启动 `scheduler.clear()` 后按 `sys_job` 重建。种子 cron 为 6 段含 `?`：`0/10 * * * * ?`（`sql:600-602`），种子 `status='1'`（暂停）。
- 方案：①`cron.New(cron.WithSeconds(), cron.WithChain(cron.SkipIfStillRunning(...)))`，入库前把 `?` 归一 `*`；②注册表 `map[string]any`（`ryTask` → `*TaskRy`），`MethodByName` 反射 + 按目标类型转换，套用同款类型推断与黑白名单；③`concurrent=1` 用 `sync.Mutex` + `SkipIfStillRunning`（等价 `@DisallowConcurrentExecution`）；④misfire 自实现（3 放弃=默认、2 启动后补跑一次、1 立即补跑）；⑤执行前后写 `sys_job_log`，异常截断 2000（对齐 `AbstractQuartzJob.java:32-95`），注意 `sys_job_log` 无 `update_by/update_time`（`sql:610-620`）；⑥`sys_job` 复合主键 `(job_id, job_name, job_group)`（`sql:597`），GORM 需三个 primaryKey、仅 `job_id` 自增；⑦`QRTZ_*` 保留 DDL 不写数据，如需任务详情页兼容只保留 `QRTZ_JOB_DETAILS/QRTZ_TRIGGERS/QRTZ_CRON_TRIGGERS` 三张结构。

### d) `@Log` + AOP → 路由元数据中间件 + 异步 worker 池
- 事实：注解 `common/annotation/Log.java:25-50`（`title/businessType/operatorType/isSaveRequestData/isSaveResponseData/excludeParamNames`）；切面 `framework/aspectj/LogAspect.java` L48 敏感字段 `{password, oldPassword, newPassword, confirmPassword}`、L54 `PARAM_MAX_LENGTH=2000`（请求/响应/异常均截断）、L101 operUrl 截断 255、L112-116 失败 `BusinessStatus.FAIL.ordinal()`、L120 `method=类名.方法名()`、L237-262 过滤 `MultipartFile/Request/Response/BindingResult`。异步 `AsyncManager.java:45` 延迟 **10ms**，`AsyncFactory.java:89-101` 在 TimerTask 内补 `operLocation` 再入库；池 `ThreadPoolConfig.java:20-61`（core50/max200/queue1000/CallerRuns）。`sys_oper_log` 无 `create_time`，用 `oper_time`（`sql:420-442`）。
- 方案：`router` 注册时附带日志元数据 → `middleware.OpLog(meta)` 用包装 `ResponseWriter` 捕获响应体（等价 `@AfterReturning(jsonResult)`）→ 组装 `SysOperLog` 投入带缓冲 channel 的 worker 池。脱敏与截断常量照搬；`operLocation` 走 `AddressUtils`（默认 `ruoyi.addressEnabled=false`，`application.yml:8`）。

### e) Excel → excelize + **struct tag 替代注解** + 反射
- 事实：`@Excel` 实测默认值（`common/annotation/Excel.java:19-196`）：`sort=MAX_VALUE, width=16, height=14, dateFormat="", readConverterExp="", dictType="", type=ALL(0)/EXPORT(1)/IMPORT(2), scale=-1, roundingMode=ROUND_HALF_EVEN, needMerge=false, isExport=true, isStatistics=false, cellType=STRING, headerBackgroundColor=GREY_50_PERCENT, headerColor=WHITE, backgroundColor=WHITE, color=BLACK, align=CENTER, separator=",", args={}, combo={}, prompt="", targetAttr="", defaultValue="", suffix=""`；**无 `handlerArgs`，实际叫 `args`**。`ExcelUtil.java`：列宽 `(width+0.72)*256`（L1203-1236，不是 +200）、`sheetSize=65536` 自动拆表（L115-117）、日期写 `parseDateToStr` 字符串 + 样式（L1265-1302）、字典经 `DictUtils` 读 Redis `sys_dict:`（L1280-1287/1523-1538）、`isStatistics` 合计行（L1566-1605）、文件名 `UUID+"_"+sheetName+".xlsx"`（L580-585）、只设 xlsx Content-Type **不设 Content-Disposition**（L1612-1615）。**ExcelUtil 内无脱敏逻辑**，`@Sensitive` 只是 Jackson 序列化器（`annotation/Sensitive.java:17-23`）。
- 方案：`excel:"name:用户名称;width:16;sort:1;dictType:sys_user_sex;readConverterExp:0=男,1=女;dateFormat:yyyy-MM-dd;type:ALL;isStatistics"`；`SXSSFWorkbook` 等价用 `excelize.NewStreamWriter`（65536 阈值对齐）；嵌套 struct 展开替代"扫描父类"；导入按标题行反查列，错误聚合后返回 **601**（前端对 601 走 warning，`request.js:76-109`）。
- **验收硬指标**：同一批数据，Java 与 Go 导出的 xlsx 逐单元格 diff，不允许"看起来一样"。

### f) Velocity 生成器 → `text/template`，前端模板原样保留
- 事实：21 个模板在 `ruoyi-generator/src/main/resources/vm`（Java 6 / Vue 9 / TS 3 / JS 1 / SQL 1 / XML 1）。选择逻辑 `VelocityUtils.java:145-193`：固定 8 个 + `crud→index`、`tree→index-tree`、`sub→index+sub-domain`；Element Plus 选 `vue/v3`，TS 版选 `vue/v3ts` 并追加 TS 模板。上下文变量 `tableName/ClassName/className/moduleName/BusinessName/businessName/packageName/author/datetime/pkColumn/columns/table/dicts/importList/permissionPrefix/colSpan/genView`（`:43-137`）。类型映射 `GenUtils.java:21-130`。表扫描排除 `qrtz_%`/`gen_%`（`GenTableMapper.xml:80-107`），列取自 `information_schema.columns`（`GenTableColumnMapper.xml:42-45`）。无 `VelocityConfig`，`VelocityInitializer.java:17-32` 只配 Classpath loader + UTF-8。
- 方案：①定界符改 `[[ ]]`（Go 的 `{{ }}` 与 Vue 模板冲突）；②**保留不动**：`vm/vue/**`(9)、`vm/ts/**`(3)、`vm/js`(1)、`vm/sql`(1)，只做 `$!{}`→`[[ ]]`、`#if/#foreach/#set`→`[[if]]/[[range]]/[[set]]` 的机械替换，产物须与 Velocity 输出逐字节一致；③**Go 侧重做** `vm/java/*`(6) → `model/repository/service/handler/dto`，`vm/xml`(1，MyBatis mapper) 由 GORM 模板取代；④Java getter 调用（`$column.javaField`）预计算进上下文 map，模板内不做方法调用；⑤zip 下载沿用 `GenController.java:256-261` 三个响应头。

### g) 三个 Filter → 三个 Gin 中间件
依据 `framework/config/FilterConfig.java` 与 `application.yml:133-147`：

| Filter | 原实现 | Go 实现 |
| --- | --- | --- |
| XSS | `common/filter/XssFilter.java:63` **GET/DELETE 放行**，L67 `excludes` 匹配；`XssHttpServletRequestWrapper.java:41/52/65` 对参数值与 JSON body 做 `EscapeUtil.clean().trim()`；`@ConditionalOnProperty("xss.enabled")` | `middleware.Xss()`：方法白名单放行 + `/system/notice` 排除 + 仅对 `/system/*,/monitor/*,/tool/*` 生效；对 query/form/JSON body 递归做等价 clean |
| Referer | `FilterConfig.java:52-66`，urlPatterns **仅 `/profile/*`**，默认 `enabled:false`；`common/filter/RefererFilter.java:44-48` **空 Referer → 403**，L52-59 子串匹配 `allowed-domains` | `middleware.Referer()`：只注册到 `/profile/*`；空 Referer 403；子串包含判断；默认关闭 |
| Repeatable | `FilterConfig.java:68-78`，`/*` 无条件注册；`common/filter/RepeatableFilter.java:32-35` **contentType 以 `application/json` 开头**才包装；`RepeatedlyRequestWrapper.java:30/42` 一次性缓存 `byte[]` | `middleware.RepeatableBody()`：`strings.HasPrefix(ctype,"application/json")` → `io.ReadAll` 后回写 `c.Request.Body` |

注册顺序：`Repeatable → Cors → Referer → Xss → JWT → Permi → (service 层 DataScope) → OpLog`。Repeatable 必须最先，否则 OpLog 读不到 body。

### h) `@DataSource` → slave 默认关闭，先单库 + 预留
依据 `common/enums/DataSourceType.java:8-18`（仅 MASTER/SLAVE）、`framework/aspectj/DataSourceAspect.java:23-70`（方法注解优先于类注解 + ThreadLocal + finally 清除）、`framework/datasource/DynamicDataSource.java:12-25`、`application-druid.yml:7-18`（`slave.enabled: false`，url/username/password 全空）、`DruidConfig.java:32-79`。
方案：一期只建 master 一个 `*gorm.DB`（连接池照搬 `initialSize 5/minIdle 10/maxActive 20/maxWait 60000`）；`pkg/db.DB(ctx)` 按 ctx 中 datasource key 选择，`slave.enabled=true` 时再建第二池（手写约 20 行，**不引入 dbresolver**）；语义对齐方法级覆盖类级 + defer 清除。

## 11. 前端契约清单（实测核对版）

> 前端根 `F=/Users/mac/Desktop/若依/RuoYi-Vue3-master`

**传输层 `F/src/utils/request.js`**：`:14-20` `application/json;charset=utf-8`、`baseURL=VITE_APP_BASE_API`、`timeout:10000`；`:24-40` `Authorization: 'Bearer ' + getToken()`，GET 的 params 经 `tansParams` 展平，**项目无 qs**；`:41-67` 1 秒内同 URL+同 JSON 重复提交前端直接 reject（`sessionObj`，≥5MB 跳过）；`:76-109` 200 通过、401 弹窗→确认后 logout + `location.href='/index'`（取消仅关弹窗，**两者最终均 reject**）、500 `ElMessage`+reject、601 warning+reject、其他非 200 `Notification`+reject、**无 code 按 200**；`:126-150` 导出走 POST + form-urlencoded + `responseType:'blob'`，**文件名由调用方传入，不解析 Content-Disposition**。

**路由 `F/src/store/modules/permission.js` + `RouterVo.java`**：`:35-52` `/getRouters` 须返回 `{data:[...]}`；`:59-83` `component` 命中 `Layout`/`ParentView`/`InnerLink` 三值，否则按 `views/**/*.vue`，空 children 时 `delete children`、`delete redirect`；`:86-97` ParentView 压平、子 path = `parent.path+'/'+child.path`。后端字段以 `ruoyi-system/.../vo/RouterVo.java:11-57` 为准（`name/path/hidden/redirect/component/query/alwaysShow/meta/children`，**`@JsonInclude(NON_EMPTY)`**），`MetaVo.java:15-30`（`title/icon/noCache/link`）；三值常量 `UserConstants.java:58-64`，决策 `SysMenuServiceImpl.java:487-501/523-525/534-536`，内链替换 `:613-616`，3.9 新增 `route_name` 列参与 `getRouteName`（`sql:135-155`）。侧边栏还依赖 `hidden/alwaysShow/path/query/meta.title/icon`（`F/src/layout/components/Sidebar/SidebarItem.vue:2-15,53-90`）、`meta.noCache`（`F/src/store/modules/tagsView.js:62-65,182-183`）、`meta.link`（`F/src/router/index.js:8-23`）。

**登录/验证码/用户**：`F/src/api/login.js:4-18` `POST /login` body `{username,password,code,uuid}`、`GET /getInfo`、`POST /logout`、`GET /captchaImage`；`F/src/views/login.vue:143-149` 读 `captchaEnabled/img/uuid`，`img` **裸 base64**（前端自拼 `data:image/gif;base64,`）；`:111-120,153-160` 记住密码 cookie `username/password/rememberMe` 30 天；`F/src/store/modules/user.js:24-85` 存 `res.token`（Cookie key `Admin-Token`，`F/src/utils/auth.js:3-14`），消费 **user/roles/permissions/pwdChrtype/isDefaultModifyPwd/isPasswordExpired**（后端 `SysLoginController.java:86-93`，`pwdChrtype` 默认 `"0"`）。验证码 Redis `captcha_codes:<uuid>` TTL 2 分钟（`Constants.java:96`），math 型 `@` 后半为答案（`CaptchaController.java:67-69`）。登录锁 `pwd_err_cnt:<username>` 5 次锁 10 分钟（`SysPasswordService.java:27-31,57-65` + `application.yml:41-46`）。

**统一返回体/分页**：`AjaxResult.java:13-24,39-59` 是 `HashMap`，`{code,msg,data}`，`data==null` 不输出，**字段顺序无契约**；码值 `HttpStatus.java:8-93`（200/401/403/500/601，**601 仅由 `AjaxResult.warn()` 产生**，`AjaxResult.java:111-126,190`）；`TableDataInfo.java:15-44` `{total,rows,code:200,msg:"查询成功"}`；入参 `TableSupport.java:16-49` + `PageDomain.java:66-99`：`pageNum`(1)/`pageSize`(10)/`orderByColumn`/`isAsc`(asc)/`reasonable`(true)，**取自 query string 或 form，不读 JSON body**；日期范围 `params[beginTime]/params[endTime]`（`F/src/utils/ruoyi.js:54-66`），**不存在 `params[dataScope]`**。

**导入导出/文件**：导出响应须带 xlsx `Content-Type` + `download-filename`(percent-encoded) + `Access-Control-Expose-Headers: Content-Disposition,download-filename`（`FileUtils.java:224-226`），前端 `F/src/plugins/download.js:13-40` **只读 `download-filename`，无回退**；文件名 `UUID+"_"+sheetName+".xlsx"`（`ExcelUtil.java:580-585`）；导入 `POST /system/user/importData?updateSupport=0|1`，字段名 `file`（`F/src/components/ExcelImportDialog/index.vue:3,69-74`）；上传 `POST /common/upload` 返回必须含 `fileName`（`F/src/components/FileUpload/index.vue:47-96,163-166`），完整字段 `url/fileName/newFileName/originalFilename`（`CommonController.java:84-89`），错误 Blob 的 type 必须是 `application/json`（`F/src/utils/ruoyi.js:225-228`）；下载 `GET /common/download`、`/common/download/resource`。

**接口清单**（`F/src/api/**`）：`/system/user`(list/{userId}/resetPwd/changeStatus/profile/profile:updatePwd/profile:avatar/authRole/{userId}/deptTree)、`/system/role`(dataScope/changeStatus/authUser/*/deptTree/{roleId})、`/system/dept`(list/exclude/{deptId}/updateSort)、`/system/menu`(treeselect/roleMenuTreeselect/{roleId}/updateSort)、`/system/dict/type`(refreshCache/optionselect)、`/system/dict/data`(type/{dictType})、`/system/config`(configKey/{key}/refreshCache)、`/system/notice`(**listTop/markRead/markReadAll/readUsers/list**，3.9 新增，见 `SysNoticeController.java:89-132`)、`/system/post`、`/monitor/{operlog,logininfor,online,job,jobLog,cache,server}`、`/tool/gen`。**不存在** `src/api/system/profile.js`、`src/api/common/upload.js`。

**数据类型契约**：状态字段是**字符串 `"0"/"1"`**（`F/src/views/system/user/index.vue:59-64`），真布尔只有 `captchaEnabled/hidden/alwaysShow/noCache/disabled/isDefaultModifyPwd/isPasswordExpired`；`userId !== 1` 判断超管（`user/index.vue:74-84`），ID 按数字返回且需落在 `2^53-1` 内；树 props `{id,label,children,disabled}`（`user/index.vue:103-105`、`role/index.vue:175-184`），`menuIds/deptIds` 含半选节点随 body 提交，授权角色用逗号串 `roleIds` 放 query（`authRole.vue:94-99`）；时间接受 `YYYY-MM-DD[ T]HH:mm:ss(.SSS)`（`F/src/utils/ruoyi.js:7-24`）。

**代理与端口**：`F/vite.config.js:5,44-59` dev port **80**，`/dev-api → http://localhost:8080`，`changeOrigin:true`，`rewrite` 去 `/dev-api`；`F/.env.development:8`=`/dev-api`，`F/.env.production:8`=`/prod-api`。**硬约束：Go 必须监听 8080**。

## 12. 三大风险与规避

**风险 1：数据权限 + 权限的隐式语义漂移（最高危，115 处 + 5 个 scope）**
Java 侧是 AOP + `PermissionContextHolder` request scope + `${}` 拼串三件套，任一环遗漏表现为"越权可见"或"列表空"，且只有非 admin 账号才暴露。
规避：①5 个 scope 与"空条件 → `dept_id = 0`"兜底写成表驱动单测，五种账号 × 用户/角色/部门三个列表做**与 Java 版逐行 diff 的 golden 测试**；②`perm_table.go` 与 115 条 Java 注解做脚本级双向 diff（缺失即 CI 失败）；③上线前用最小权限账号走一遍全菜单。

**风险 2：Excel / 代码生成器的反射元数据对等性**
`ExcelUtil.java` 71KB、`@Excel` 30+ 属性、21 个 Velocity 模板；Go 无注解无 Velocity，最容易"能用但差一点"（列宽/日期/字典/合计行/前端模板产物），而前端不做容错。
规避：①导出产物与 Java 版**逐单元格 diff**回归；②`vm/vue/**`、`vm/ts/**`、`vm/js`、`vm/sql` 只做定界符机械替换，产物**逐字节 diff**，禁止顺手美化；③定界符统一 `[[ ]]`，避免与 Vue 的 `{{ }}` 冲突。

**风险 3：写入侧语义差异 + 前端隐性契约**
`sql/` 全库**无 `ON UPDATE CURRENT_TIMESTAMP`、无 `CURRENT_TIMESTAMP` 默认值**（grep 零命中），`create_time/update_time` 全靠应用层写；`del_flag` 为 `'0'/'2'` 三态字符串（GORM `DeletedAt` 语义相反，不可直接复用）；`sys_job` 复合主键、`sys_job_log` 无 `update_time`、4 张关联表无任何时间字段；`sys_dict_type.dict_type` 与 `sys_notice_read(user_id, notice_id)` 唯一约束；`char(1)`/`int` 的 0/1 标记建议统一建模为 `string`。前端侧隐性依赖：`download-filename`、601、1 秒防重、`Bearer ` 前缀、8080 端口。
规避：①统一 BaseModel 用 `*time.Time` + `BeforeCreate/BeforeUpdate` hook 显式填时间，`del_flag` 用 `string` + 常量 `Exists="0"/Deleted="2"`，**且必须为这两项各写表驱动单测，与权限/数据权限单测同级别必要**；②建库显式 `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`（原 SQL 未声明字符集，`sys_notice.notice_content` 为 longblob 正文）；③维护"前端契约测试矩阵"（登录、getInfo 六字段、getRouters 三值、导出头、上传返回、401/500/601 各一例），每次发版回归。
