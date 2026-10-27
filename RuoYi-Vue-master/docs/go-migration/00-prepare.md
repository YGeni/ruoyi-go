# Go（Gin）迁移准备工作

在把 `docs/go-migration/` 里的提示词喂给 Codex 之前，先跑上面的脚本完成环境准备。

## 一键完成（推荐）

```bash
cd /Users/mac/Desktop/若依/RuoYi-Vue-master
./server-go/scripts/setup.sh
```

脚本幂等，重复执行安全，依次完成：校验 Docker → 克隆前端契约基准到 `/tmp/ruoyi-ui-ref` → 起 MySQL+Redis 容器 → 等待健康检查 → 校验 `ry-vue` 库是否已有 31 张表（不足则从 `sql/quartz.sql` + `sql/ry_20260417.sql` 导入）→ 打印下一步。

停止环境：`docker compose -f server-go/docker-compose.yml down`

## 手工步骤（脚本的等价展开）

### 1. 前端契约基准（在本机，无需克隆）

前端仓库 `/Users/mac/Desktop/若依/RuoYi-Vue3-master`（RuoYi-**Vue3**）与后端同级相邻，
`setup.sh` 会校验它存在；路径不同时用 `UI_REF=/your/path ./scripts/setup.sh` 覆盖。

**注意**：脚本不再联网克隆Vue2 版仓库——Vue2 的 `getRouters` 返回结构与 Vue3 不同，
拿它做契约基准会导致第 1 期验收对不上。一切契约以本仓库 Vue3 前端源码为准。

### 2. 启动本地依赖

迁移过程需要真实跑冒烟（登录 → token → 调列表），不能只靠静态检查。

```bash
cd /Users/mac/Desktop/若依/RuoYi-Vue-master
docker compose -f server-go/docker-compose.yml up -d
```

凭据与源文件保持一致（`ruoyi-admin/src/main/resources/application-druid.yml:7-18`）：

- MySQL `localhost:3306`，`root` / `password`，库 `ry-vue`
- Redis `localhost:6379`，`db0`（`application.yml:84-93`）

数据由脚本从 `sql/quartz.sql` + `sql/ry_20260417.sql` 初始化，**不要改表结构**。

### 3. 确认 Go 工程目录

Go 工程位于仓库根的 `server-go/`（已含 `docker-compose.yml` 与 `scripts/`），拥有独立 `go.mod`；Maven 不会理会该目录，两套构建互不干扰。

## 4. 使用方式（Codex CLI）

- **一期一个会话**：不要把全流程塞进一次对话；上下文过半就开新会话，让它先重读根目录 `AGENTS.md`。
- **审批模式优先**：跑整期时不要用全自动模式，删除/覆盖文件前的暂停正好当作人工卡点。
- **每期收尾**：要求 Codex 写出「本期遗留项」，直接作为下一期的输入。
- 提示词里的路径写绝对路径，不要写「参考若依」这类虚指——Codex 走真实文件读取，路径具体远比描述准确。

## 阶段划分

| 期 | 目标 | 产出验证 |
| --- | --- | --- |
| 第 0 期 | 方案评审（不写代码） | 目录结构、依赖清单、三期实施计划、8 个机制替代方案 |
| 第 1 期 | 骨架 + 配置 + 认证授权 | 前端可登录、可拿到菜单路由 |
| 第 2 期 | system 其余模块 + 日志 + 监控 + 文件 | 各列表/增删改查与 Java 版对等 |
| 第 3 期 | Excel + 定时任务 + 代码生成器 | 导入导出、Job 调度、生成 Go/Vue 代码 |

> 第 3 期的代码生成器放最后：它依赖前两期所有模块字段规范定稿，提前写必然返工。
