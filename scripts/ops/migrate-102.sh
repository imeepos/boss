#!/usr/bin/env bash
# 102 迁移执行/回滚(可验证回滚路径的落地工具):
#   scripts/ops/migrate-102.sh status              当前已应用版本
#   scripts/ops/migrate-102.sh up    [N]           应用 N 个待应用迁移(默认全部)
#   scripts/ops/migrate-102.sh down  <VERSION>     回滚到指定版本(执行其后所有 .down.sql,新->旧)
# down 前自动做一次备份(backup-102.sh),失败即中止。
# 语义与 internal/pkg/database.Migrate 对齐:文件自带 BEGIN/COMMIT,成功后记账 schema_migrations。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
MIG="$ROOT/migrations"
SSH_HOST="${SSH_HOST:-imeepos@192.168.0.102}"
PG_CONTAINER="${PG_CONTAINER:-boss-infra-postgres-1}"
PG_USER="${PG_USER:-boss}"
PG_DB="${PG_DB:-boss}"

psql_remote() {  # stdin=SQL;-tA 无装饰输出,grep -qx 精确匹配用
  ssh "$SSH_HOST" "docker exec -i $PG_CONTAINER psql -U $PG_USER -d $PG_DB -v ON_ERROR_STOP=1 -q -tA"
}

applied_versions() { psql_remote <<< "SELECT version FROM schema_migrations ORDER BY version;"; }

run_sql_on_102() {  # $1=本地sql文件
  ssh "$SSH_HOST" "docker exec -i $PG_CONTAINER psql -U $PG_USER -d $PG_DB -v ON_ERROR_STOP=1 -q" < "$1"
}

APPLIED_CACHE=""
is_applied() {  # 惰性缓存已应用列表,避免逐迁移开 ssh
  [ -z "$APPLIED_CACHE" ] && APPLIED_CACHE=$(applied_versions)
  printf '%s\n' "$APPLIED_CACHE" | grep -qx "$1"
}

case "${1:-status}" in
status)
  echo "== 已应用 =="; applied_versions
  echo "== 待应用 =="
  for f in "$MIG"/*.up.sql; do
    v=$(basename "$f" .up.sql)
    is_applied "$v" || echo "$v"
  done
  ;;
up)
  n="${2:-999}"
  count=0
  for f in "$MIG"/*.up.sql; do
    [ "$count" -ge "$n" ] && break
    v=$(basename "$f" .up.sql)
    if is_applied "$v"; then continue; fi
    echo "apply $v"
    run_sql_on_102 "$f"
    psql_remote <<< "INSERT INTO schema_migrations(version) VALUES('$v');"
    count=$((count+1))
  done
  [ "$count" -gt 0 ] || echo "无可应用迁移"
  echo "applied $count"
  ;;
down)
  target="${2:?用法: migrate-102.sh down <回滚到的版本,如 000111_cdr_kafka_status>}"
  echo "== down 前强制备份 =="
  "$ROOT/scripts/ops/backup-102.sh"
  # 收集已应用且 > target 的版本,新->旧逐个回滚
  to_rollback=$(applied_versions | awk -v t="$target" '$0 > t' | sort -r)
  [ -n "$to_rollback" ] || { echo "无需回滚(目标即当前状态)"; exit 0; }
  for v in $to_rollback; do
    f="$MIG/$v.down.sql"
    [ -f "$f" ] || { echo "缺少 $v.down.sql,中止"; exit 1; }
    echo "revert $v"
    run_sql_on_102 "$f"
    psql_remote <<< "DELETE FROM schema_migrations WHERE version='$v';"
  done
  echo "rolled back to $target"
  ;;
*)
  echo "用法: $0 {status|up [N]|down <VERSION>}" >&2; exit 2 ;;
esac
