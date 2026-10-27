# 第 0 期：方案评审提示词（交给 Codex，不写代码）

复制下方整段代码块作为新会话的第一条消息。

```text
你是 Go 后端架构师。任务：把本机 /Users/mac/Desktop/若依/RuoYi-Vue-master（RuoYi-Vue 3.9.2，
Spring Boot + Java 17，Maven 多模块）的后端渐进式重构为 Go 项目，代码写在 server-go/，
保留原 ruoyi-* 六个模块不动，禁止删除或修改现有 Java 代码。

技术栈（已定，不要替换）：Gin + GORM + Casbin + golang-jwt/v5 + go-redis/v9 +
robfig/cron/v3 + excelize + viper + zap + go-playground/validator + base64Captcha。
遇到与 gin-vue-admin 同构的问题，优先采用其成熟做法。

【现状，请以实际读文件为准】
- 模块职责：ruoyi-admin（21 个 Controller，全部 HTTP 入口）、ruoyi-framework（Security/JWT/AOP/Redis/
  线程池）、ruoyi-system（13 Service + 16 Mapper，业务主体）、ruoyi-common（工具类，
  ExcelUtil.java 71KB）、ruoyi-quartz（表驱动定时任务）、ruoyi-generator（Velocity 生成器，21 个模板）
- 数据库：sql/ry_20260417.sql（20 张业务表）+ sql/quartz.sql（11 张 Quartz 表），共 31 张。
  无存储过程、无触发器、无视图、无 WebSocket，动态 SQL 复杂度低 → 这是迁移的低风险面
- 配置基线：ruoyi-admin/src/main/resources/application.yml + application-druid.yml
  （端口 8080、context-path /、MySQL ry-vue、Redis localhost:6379 db0、token 有效期 30 分钟、
  登录失败 5 次锁 10 分钟、单文件 10MB）

【本轮只做方案评审，不要写任何代码】输出以下内容并用实际文件路径佐证：
1. server-go/ 的目录结构与分层（handler / service / model / middleware / router / config / pkg），
   说明它如何对应原六个模块的职责
2. go.mod 依赖清单，逐个标注用途
3. 三期实施计划，每期列产出物、验收标准、可以和前端联调的功能点
4. 下列 8 个 Java 特有机制在 Go 侧的替代方案，必须逐个给出，不能含糊：
   a) @PreAuthorize("@ss.hasPermi('system:user:list')") 方法级鉴权，全仓库约 116 处，
      roleKey=admin 直通 → Casbin 还是自写权限中间件 + Redis 权限集，给出理由
   b) @DataScope 数据权限：SysUserServiceImpl 3 处、SysRoleServiceImpl、SysDeptServiceImpl 各 1 处，
      现有实现是 AOP 生成 SQL 片段后用 ${params.dataScope} 拼进 SysUserMapper.xml 第 86/103/121 行
   c) Quartz 表驱动调度：sys_job.invoke_target 形如 "ryTask.ryParams('ry')"，靠 Java 反射调用 Bean
   d) @Log 注解 + AOP 异步落 sys_oper_log
   e) POI + 注解反射的 Excel 导出/导入（保留 @Excel 的列宽、日期格式、字典回写、脱敏语义）
   f) Velocity 代码生成器（后端模板需在 Go 侧重做，vm/vue/* 与 vm/ts/* 前端模板必须保留）
   g) XSS Filter（排除 /system/notice）、Referer 防盗链、Repeatable Body 过滤器
   h) @DataSource 动态数据源（注意 slave 默认 enabled:false）
5. 必须与原前端保持一致的契约清单。前端源码在本机 /Users/mac/Desktop/若依/RuoYi-Vue3-master
   （RuoYi-**Vue3**，不是 Vue2），请实际读它的源码后列出，重点核对以下已确认的锚点：
   - src/utils/request.js:78-109 —— code 语义：200 成功、401 会话失效跳登录、
     500 错误提示、601 警告；非 200 一律 reject
   - src/utils/request.js:32 —— 请求头 Authorization 带 "Bearer " 前缀，
     Go 侧必须剥离前缀再验签（对应原实现 ruoyi-framework/.../web/service/TokenService.java:219-227）
   - src/store/modules/permission.js:35-97 —— getRouters 的返回值由它解析：
     component 取值 'Layout' / 'ParentView' / 'InnerLink' 或 views 相对路径；
     叶子节点会被 delete children 与 redirect
   - src/views/login.vue:143-150 —— 验证码接口返回 { captchaEnabled, img, uuid }，
     img 是不含 "data:image/gif;base64," 前缀的裸 base64，前端自行拼接
   - vite.config.js:5,48-54 —— 前端 dev 代理目标是 http://localhost:8080 并去掉 /dev-api 前缀，
     Go 后端若改用其他端口需同步改这里
   - 统一返回体 AjaxResult、列表返回体 TableDataInfo、分页入参、导入导出的文件名与 Content-Type
6. 三条你认为最大的技术风险，以及规避手段

格式要求：先用 300 字以内给结论，再展开。所有结论必须标注来源文件路径，
不允许基于"若依的标准做法"这类外部记忆做判断，一律以本仓库代码为准。
```
