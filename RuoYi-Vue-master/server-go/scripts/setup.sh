#!/usr/bin/env bash
#
# Go(Gin) 迁移前置准备：拉前端契约基准 + 起 MySQL/Redis + 校验数据初始化
# 幂等：重复执行安全。任意项目位置均可运行。
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
SERVER_GO="${REPO_ROOT}/server-go"
COMPOSE_FILE="${SERVER_GO}/docker-compose.yml"
UI_REF="/tmp/ruoyi-ui-ref"
EXPECT_TABLES=31

info() { printf '\033[36m[info]\033[0m %s\n' "$*"; }
ok()   { printf '\033[32m[ok]\033[0m %s\n' "$*"; }
err()  { printf '\033[31m[err]\033[0m %s\n' "$*" >&2; }

command -v docker >/dev/null 2>&1 || { err "未检测到 docker，请先安装 Docker Desktop"; exit 1; }
docker info >/dev/null 2>&1       || { err "docker 守护进程未运行，请先启动 Docker Desktop"; exit 1; }

# ---- 1. 前端契约基准 ----
info "1/4 准备前端契约基准 -> ${UI_REF}"
if [[ -d "${UI_REF}/ruoyi-ui/src" ]]; then
  ok "已存在，跳过克隆"
else
  rm -rf "${UI_REF}"
  git clone --depth=1 https://github.com/yangzongzhuan/RuoYi-Vue.git "${UI_REF}"
  ok "克隆完成"
fi

# ---- 2. 启动依赖 ----
info "2/4 启动 MySQL + Redis"
docker compose -f "${COMPOSE_FILE}" up -d
ok "容器已启动"

# ---- 3. 等待健康检查 ----
wait_healthy() {
  local name="$1" i
  info "3/4 等待 ${name} 就绪"
  for i in $(seq 1 60); do
    status="$(docker inspect -f '{{.State.Health.Status}}' "${name}" 2>/dev/null || echo missing)"
    [[ "${status}" == "healthy" ]] && { ok "${name} 就绪"; return 0; }
    sleep 2
  done
  err "${name} 健康检查超时"; docker logs --tail 30 "${name}" >&2; exit 1
}
wait_healthy ruoyi-go-mysql
wait_healthy ruoyi-go-redis

# ---- 4. 校验 / 导入数据 ----
info "4/4 校验 ry-vue 库的数据初始化"
count_sql="SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='ry-vue';"
count="$(docker exec -i ruoyi-go-mysql mysql -uroot -ppassword -N -B -e "${count_sql}" 2>/dev/null | tr -d '[:space:]')"
count="${count:-0}"

if [[ "${count}" -lt "${EXPECT_TABLES}" ]]; then
  info "表数量 ${count}/${EXPECT_TABLES}，导入 sql 脚本"
  docker exec -i ruoyi-go-mysql mysql -uroot -ppassword ry-vue < "${REPO_ROOT}/sql/quartz.sql"
  docker exec -i ruoyi-go-mysql mysql -uroot -ppassword ry-vue < "${REPO_ROOT}/sql/ry_20260417.sql"
  count="$(docker exec -i ruoyi-go-mysql mysql -uroot -ppassword -N -B -e "${count_sql}" | tr -d '[:space:]')"
fi

if [[ "${count}" -lt "${EXPECT_TABLES}" ]]; then
  err "表数量 ${count}/${EXPECT_TABLES}，数据未完整初始化，请检查 sql/ 目录与容器日志"
  exit 1
fi
ok "数据库就绪，共 ${count} 张表"

cat <<EOF

准备就绪：
  MySQL   localhost:3306  root/password  db=ry-vue  (${count} 张表)
  Redis   localhost:6379  db0
  前端契约基准  ${UI_REF}/ruoyi-ui/src

下一步：把 ${REPO_ROOT}/docs/go-migration/01-review-prompt.md 中的代码块
复制给 Codex，开启第 0 期方案评审。

停止环境：docker compose -f ${COMPOSE_FILE} down
EOF
