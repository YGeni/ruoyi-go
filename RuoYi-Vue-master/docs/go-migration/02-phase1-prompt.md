# 第 1 期：骨架 + 认证授权落地提示词

方案评审通过之后，**开一个新会话**，把下面这段粘进去。

```text
按上一轮确认的方案，实现第一期：项目骨架 + 配置 + 认证授权 + 用户/角色/菜单/字典。

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

【本期产出】
1. go.mod、目录骨架、viper 配置（严格对齐 application.yml + application-druid.yml，Redis key 名沿用）
2. 验证码 / 登录 / 退出 / 获取用户信息 / getRouters 五个接口
3. JWT 签发与自动刷新，登录态存 Redis，key 沿用 login_tokens:
4. 权限中间件替代 @PreAuthorize hasPermi / hasAnyPermi / hasRole / hasAnyRoles，admin 直通
5. 密码 BCrypt + Redis 登录失败计数（pwd_err_cnt:，5 次锁 10 分钟）
6. resultMap 的 association/collection（user→dept、user→roles）用 GORM Preload 实现

【硬性约束】
- 返回体字段名与语义必须能被 /tmp/ruoyi-ui-ref/ruoyi-ui 直接消费，禁止自定义 code 值
- 禁止字符串拼接 SQL：所有动态条件、数据权限片段一律用 GORM 构造器参数化
- find_in_set(ancestors)、date_format、sysdate() 等 MySQL 专有写法改写为等价实现，
  并在改动处注明原语义
- 与原实现有行为差异时，在代码里留 // DIFF(review): 注释并说明理由，禁止静默偏离
- 不改动任何 SQL 表结构

【流程与验收】
- 先列出本次要创建的文件清单，等我确认后再逐个实现，每完成一个文件同步进度
- 完成后依次执行 go build ./... 和 go vet ./...，必须零报错零告警
- 为权限中间件和数据权限各写一个表驱动单测，覆盖 admin 直通、跨部门数据隔离两个场景
- 给出可实际执行的 curl 冒烟序列（取验证码 → 登录 → 带 token 调 getRouters），
  并真实跑一遍贴出结果。跑不通的接口如实说明，不要谎称通过
- 全部验证通过后 git commit 一次，commit message 说明本期范围与遗留项
```
