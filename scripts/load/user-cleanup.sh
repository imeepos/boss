#!/usr/bin/env bash
# user 压测造数清理: 删除 requestId LIKE 'perf-%' 的历史订单(下单环节1 PENDING,无下游派单),
# 恢复地址在途额度,保证"造数不过夜"。默认 dry-run 只计数;加 --apply 才执行。
# 用法: scripts/load/user-cleanup.sh [--apply]
set -uo pipefail
MODE="${1:-dry-run}"
SSH_HOST="imeepos@192.168.0.102"
PG="docker exec -i boss-infra-postgres-1 psql -U boss -d boss"

echo "== user-cleanup ($MODE) =="
COUNT=$(ssh -o BatchMode=yes -o ConnectTimeout=5 "$SSH_HOST" \
  "$PG -tAc \"SELECT count(*) FROM orders WHERE request_id LIKE 'perf-%'\"")
echo "perf- 订单数: $COUNT"
[ "$COUNT" = "0" ] && { echo "无 perf- 造数, 无需清理"; exit 0; }

if [ "$MODE" = "--apply" ]; then
  ssh -o BatchMode=yes "$SSH_HOST" "$PG -v ON_ERROR_STOP=1" <<'SQL'
BEGIN;
DELETE FROM order_stages WHERE order_id IN (SELECT id FROM orders WHERE request_id LIKE 'perf-%');
DELETE FROM orders WHERE request_id LIKE 'perf-%';
COMMIT;
SQL
  rc=$?
  [ $rc -eq 0 ] && echo "清理完成" || echo "清理失败(已回滚)"
  exit $rc
else
  echo "dry-run: 未执行; 加 --apply 清理"
  exit 0
fi