# 第 1 期：骨架 + 认证授权落地提示词

方案评审通过之后，**开一个新会话**，把下面这段粘进去。

```text
按 AGENTS.md §7–§12（方案评审定稿，必读）确认的方案，实现
第 1 期中的 **1a 步**（见 §9"实施顺序"）：项目骨架 + 配置 + 认证授权 + 登录/菜单/验证码。

范围严格限定为 1a，**不要写 1b**（数据权限、各模块 CRUD、文件上传）的内容。
目标：前端能用 admin/admin123 登录成功，并看到完整菜单。

【严格限定范围，超出部分不要写】
参照实现（自行读取）：
- ruoyi-framework/src/main/java/com/ruoyi/framework/config/SecurityConfig.java
- ruoyi-framework/src/main/java/com/ruoyi/framework/web/service/TokenService.java
- ruoyi-framework/src/main/java/com/ruoyi/framework/web/service/PermissionService.java
- ruoyi-framework/src/main/java/com/ruoyi/framework/web/service/SysPermissionService.java
- ruoyi-framework/src/main/java/com/ruoyi/framework/web/service/SysPasswordService.java
- ruoyi-system/src/main/resources/mapper/system/ 下的 SysUserMapper.xml、SysMenuMapper.xml、SysRoleMapper.xml
- ruoyi-admin/src/main/java/com/ruoyi/web/controller/system/SysLoginController.java
- ruoyi-admin/src/main/java/com/ruoyi/web/controller/system/SysMenuController.java（getRouters）
- 前端契约：/Users/mac/Desktop/若依/RuoYi-Vue3-master/src 下的
  utils/request.js、store/modules/permission.js、views/login.vue、api/login.js、api/menu.js

【本期产出】
1. go.mod（**语言版本必须写 go 1.25**，本机实测 go1.26.5，禁止写 1.22）、目录骨架、
   viper 配置（严格对齐 application.yml + application-druid.yml，Redis key 名沿用）
2. 验证码 / 登录 / 退出 / 获取用户信息 / getRouters 五个接口
3. JWT 签发与自动刷新，登录态存 Redis，key 沿用 login_tokens:
   校验时必须剥离 Authorization 头的 "Bearer " 前缀（前端恒定带前缀，见 src/utils/request.js:32；
   原实现见 ruoyi-framework/.../web/service/TokenService.java:219-227）
4. 权限中间件替代 @PreAuthorize hasPermi / hasAnyPermi / hasRole / hasAnyRoles，admin 直通
5. 密码 BCrypt + Redis 登录失败计数（pwd_err_cnt:，5 次锁 10 分钟）
6. resultMap 的 association/collection（user→dept、user→roles）用 GORM Preload 实现

【硬性约束】
- 服务端口必须是 8080，与 vite.config.js:5 的代理目标一致
- 返回体字段名与语义必须能被本机 RuoYi-Vue3 前端直接消费，禁止自定义 code 值
- 验证码返回 { captchaEnabled, img, uuid }，img 为不含 data URI 前缀的裸 base64
- getRouters 的字段必须满足 src/store/modules/permission.js:59-97 的解析方式
- 禁止字符串拼接 SQL：所有动态条件、数据权限片段一律用 GORM 构造器参数化
- find_in_set(ancestors)、date_format、sysdate() 等 MySQL 专有写法改写为等价实现，
  并在改动处注明原语义
- 与原实现有行为差异时，在代码里留 // DIFF(review): 注释并说明理由，禁止静默偏离
- 不改动任何 SQL 表结构

【流程与验收】
- 先列出本次要创建的文件清单，等我确认后再逐个实现，每完成一个文件同步进度
- 完成后依次执行 go build ./... 和 go vet ./...，必须零报错零告警
- 为**权限中间件**写表驱动单测，必须覆盖三种情形：普通权限命中、`*:*:*` 直通、
  无权限返回 403 且 msg 为"没有权限，请联系管理员授权"
- 为 **BaseModel 时间填充（BeforeCreate/BeforeUpdate hook）与 del_flag 常量**
  各写单测（原库所有表无 DB 默认时间，且 del_flag 是 '0'/'2' 三态字符串，见 §12 风险 3）
- 数据权限单测不在本期，留给 1b
- 给出可实际执行的 curl 冒烟序列（取验证码 → 登录 → 带 token 调 getRouters），
  并真实跑一遍贴出结果。跑不通的接口如实说明，不要谎称通过
- 全部验证通过后 git commit 一次，commit message 说明本期范围与遗留项
```
