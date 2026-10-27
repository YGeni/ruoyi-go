# 第 1 期 1b：分页 + 用户 CRUD + 数据权限 + 其余模块

1a（`35748a9`）已验收。**开新会话**使用本文提示词；先把方块代码块整体复制到剪贴板：

```bash
cd /Users/mac/Desktop/若依/RuoYi-Vue-master && awk '/^```/{f=!f; next} f' docs/go-migration/02b-phase1b-prompt.md | pbcopy
```

```text
先读本仓库根目录 AGENTS.md：重点 §7–§12（方案评审定稿）与末尾「第 1 期 1a 已交付」一节，
再开工。

本轮实现第 1 期 **1b**（范围见 AGENTS.md §9 与文末「已交付」节），即：
pkg/page + 用户 CRUD + 数据权限 @DataScope + 其余系统模块 + 文件上传下载 + profile。

【开工前置条件 —— 必须先做，不满足就停下来报告】
1. 修复种子数据乱码：Docker 初始化 SQL 时客户端字符集为 latin1，导致 sys_menu.menu_name、
   sys_dict_data.dict_label 等中文被二次编码入库。请按 AGENTS.md 末尾给出的命令重导数据
   （DROP + CREATE DATABASE ... utf8mb4，再用 --default-character-set=utf8mb4 重新导入两个 sql）。
   重导后自检：select menu_name, hex(menu_name) from sys_menu limit 1;
   必须显示正常中文且 hex 以 E7B3BB 开头（不是 C3A7C2B3）。
2. 重跑 1a 冒烟确认未被破坏（captchaImage → login → getInfo → getRouters）。
若这两步没通过，不要往下写业务代码。

【实施顺序 —— 硬约束，不许跳序】
第 1 步：pkg/page（对齐 TableSupport.java:16-49 + PageDomain.java:66-99：
        pageNum/pageSize/orderByColumn/isAsc/reasonable，只从 query/form 取，不读 JSON body）
        + internal/model/entity 补齐 role/dept/post/dict/config/notice（若已存在于 sys_misc.go 则复用）
        + handler/system/user.go 的 list + service + repository
        + 验收：GET /system/user/list 返回 TableDataInfo{total,rows,code:200,msg:"查询成功"}，
          admin 登录后能看到全部用户，分页/排序/searchValue/deptId 生效
第 2 步：数据权限 @DataScope —— 这是最高风险项，必须紧接着做，不许留到最后
        · 用 GORM Scopes 实现：DataScope(ctx, u *LoginUser, aliasDept, aliasUser) func(*gorm.DB) *gorm.DB
        · 五个分支与"空条件兜底 OR d.dept_id = 0"必须完整复刻（DataScopeAspect.java L50/L92-135）
        · find_in_set(?, ancestors) 参数化；全部条件用 ? 占位，禁止任何字符串拼接 SQL
        · permission 由 Permi 中间件写入 context（等价 Java 的 request-scoped PermissionContextHolder）
        · 落点：selectUserList / selectAllocatedList / selectUnallocatedList（deptAlias=d, userAlias=u）、
          SysRoleServiceImpl:54、SysDeptServiceImpl:45
        · 表驱动单测必须覆盖：admin、自定义(2)、本部门(3)、本部门及以下(4)、仅本人(5)、
          以及"角色无 data_scope 命中时的兜底 0 行"
        · golden 验证：用 SQL 直算每种账号的期望行数，与 Go 返回的 total 逐一比对，
          差异必须列出，不允许"看起来差不多"
第 3 步：router/perm_table.go 落地 —— 把 115 处权限串绑定到路由，并做双向 diff：
        从 Java 侧提取 @PreAuthorize("@ss.hasPermi('xxx')") 全量清单，
        与 perm_table.go 登记的路由-权限做双向比对，输出差异报告（缺失/多余各列出），
        报告要能作为 CI 断言使用。
第 4 步：其余模块 role / dept / post / dict_type / dict_data / config / notice，
        含 §11 列出的 3.9 新增接口 notice 的 listTop/markRead/markReadAll/readUsers/list
第 5 步：文件上传下载 handler/common/* + profile（头像 / 改密码 / 个人信息）
        · /common/upload 返回必须含 fileName，完整字段 url/fileName/newFileName/originalFilename
        · 错误响应的 Content-Type 必须是 application/json（前端 F/src/utils/ruoyi.js:225-228）

【硬性约束】
- 端口 8080；所有 char(1) 建模为 string（role.data_scope/status、user.status/del_flag 等），
  tinyint(1) 可用 bool（role.menu_check_strictly / dept_check_strictly）
- 登录名字段是 user_name（不存在 login_name）
- 不得使用 GORM DeletedAt 实现 del_flag，一律 BaseModel 常量 Exists="0"/Deleted="2"
- 需要新依赖（如 testify）或新增文件超过 30 个时，停下来报告，不得自行扩张
- 行为差异一律留 // DIFF(review): 注释，禁止静默偏离

【流程与验收】
- 先列出本轮要创建/修改的文件清单，等我确认后再逐个实现，每完成一个文件同步进度
- gofmt -l . 必须为空、go build ./... 与 go vet ./... 必须零输出
- 单测必跑：数据权限五分支 + 兜底、分页、权限中间件（含 *:*:* 与 admin 直通）
- curl 冒烟必须包含：admin 全量列表、非 admin 账号列表行数差异、分页参数、
  401/403 各一例；用 Docker 里的真实库跑，贴出真实输出
- 跑不通的如实说明，不得谎称通过
- 全部通过后 git commit 一次，message 写明本期范围与遗留项
```

## 我这边的验收口径（把结果贴回来时对齐这 6 条）

1. 数据重导后 `getRouters` 的 `meta.title` 是**正常中文**（不是乱码）
2. `/system/user/list` 在 admin 与普通账号下返回的 `total` 与 SQL 直算结果**数值一致**
3. 数据权限五分支 + 兜底 **6 个用例全绿**，且全部参数化无字符串拼 SQL
4. `perm_table.go` 与 Java 115 处注解的**双向 diff 报告**已产出，无遗漏
5. `gofmt / build / vet / test` 四道全过
6. 冒烟输出是**真实 curl 结果**而非描述
