# 后续模块通用提示词模板

第 2、3 期逐个模块推进时套用本模板，只需替换 `{...}` 占位符。建议**一个模块一个新会话**，避免上下文膨胀。

```text
接着上一轮进度，实现 {模块名} 模块。

【对照实现，自行读取】
- 接口：ruoyi-admin/src/main/java/com/ruoyi/web/controller/system/{Xxx}Controller.java
- 逻辑：ruoyi-system/src/main/java/com/ruoyi/system/service/impl/{Xxx}ServiceImpl.java
- SQL：ruoyi-system/src/main/resources/mapper/system/{Xxx}Mapper.xml

【要求】
- {列出该模块特有机制，示例：selectUserList / selectAllocatedList / selectUnallocatedList
  三处 @DataScope，用一期确定的 Scope 方案实现，禁止拼接 SQL 片段}
- 分页用一期统一的 Paginate（替代 PageHelper 的 16 处 startPage 调用）
- 树查询的 ancestors 过滤改写后需与原 find_in_set 语义等价，补单测证明
- 现有 @Log 改为一期确定的钩子；导入导出路径、文件名、Content-Type 保持与原前端一致
- 空值处理、排序稳定性、字段命名与原实现逐项对齐，有差异写 // DIFF(review): 注释说明

完成后：go build ./... && go vet ./... → 跑 curl 冒烟 → 报告结果与实际通过的用例数。
跑不通的如实说明，并在结尾列出本期遗留项作为下一期的输入。
```

## 各模块要点速查

| 模块 | 需特别交代给 Codex 的机制 |
| --- | --- |
| 用户 `SysUserController` | 3 处数据权限、角色/岗位关联、导入导出、重置密码、状态变更 |
| 角色 `SysRoleController` | `SysRoleServiceImpl:54` 数据权限、菜单分配（role_menu）、部门范围（role_dept） |
| 部门 `SysDeptController` | `SysDeptServiceImpl:45` 数据权限、ancestors 树形维护 |
| 菜单 `SysMenuController` | getRouters 的树结构组装（一期已做）、菜单按角色过滤 |
| 字典 `SysDictTypeController` | 字典缓存 Redis `sys_dict:`，失效与刷新时机 |
| 参数 `SysConfigController` | 配置缓存 Redis `sys_config:`，加载优先级：缓存 → DB |
| 通知 `SysNoticeController` | XSS 过滤器对该路径豁免 |
| 日志 `SysOperlogController` / `SysLogininforController` | 异步落库、清理策略 |
| 监控 `monitor/*` | CPU/内存/磁盘/Redis 指标采集，无 Java oshi，用 gopsutil |
| 在线用户 `SysUserOnlineController` | 强退即删 Redis `login_tokens:` |

## 收尾第 3 期要点

- **定时任务**：`sys_job` 表保留，调度引擎换成 `robfig/cron/v3` + 显式注册表 `map[string]JobFunc`，
  **不要**尝试用反射按字符串调用（Go 无等价能力）
- **Excel**：`excelize` + struct tag 反射，复刻 `@Excel` 的列宽、日期格式、字典回写、脱敏语义
- **代码生成器**：`text/template` 重写后端模板；`vm/vue/*`、`vm/ts/*` 前端模板原样保留，
  `gen_table` / `gen_table_column` 表结构不变
