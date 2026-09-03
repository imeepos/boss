#!/usr/bin/env bash
# T17 机械验收:order_stages.finished_at 写入口径修复 + 历史回填 + 102 现网抽查。
# 四段断言,任一不过退出 1:
#   1) 静态:写入点覆盖(推进函数均含 finished_at 写入);
#   2) 静态:回填脚本存在且幂等标记(finished_at IS NULL 守卫);
#   3) SQL 抽查 102:DONE 行 finished_at 非空比例提升(基线 5/210≈2.4%,回填后≥25%);
#   4) live probe:102 现网新建 acc_ 订单 → 环节1 必须已写 finished_at(新推进路径生效),
#      验证后按 acc_ 标记清理(造数不过夜)。risk.direct.enabled 临时关停防风控误拦,结束恢复。
# 环境: SSH_HOST/BASE_URL 可覆盖;缺省 102。
set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SSH_HOST="${SSH_HOST:-imeepos@192.168.0.102}"
BASE_URL="${BASE_URL:-http://192.168.0.102:28080}"
PG="docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tA"
sql102() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "$PG"; }
KEY="${ADMIN_API_KEY:-$(python3 -c "import json;print(json.load(open('$ROOT/.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key'])" 2>/dev/null || true)}"
fail=0
fail_step() { echo "FAIL $1"; fail=1; }

# 1) 写入点覆盖:appendStage 的 INSERT 必须带 finished_at 列 + DONE 守卫;
#    pg_check 自愈 UPDATE 必须仍写 finished_at;Submit 环节1 必须落 DONE(下单即完成)。
ADV="$ROOT/internal/domain/order/pg_advance.go"
CHK="$ROOT/internal/domain/order/pg_check.go"
SUB="$ROOT/internal/domain/order/pg_submit.go"
if grep -q 'finished_at' "$ADV" && grep -q 'CASE WHEN \$3 = .DONE. THEN now() END' "$ADV"; then
  echo "PASS appendStage writes finished_at on DONE (推进成功即写)"
else
  fail_step "appendStage finished_at write coverage"
fi
if grep -q 'finished_at=now()' "$CHK"; then
  echo "PASS pg_check self-heal UPDATE keeps finished_at=now()"
else
  fail_step "pg_check self-heal finished_at"
fi
if grep -q 's.appendStage(ctx, o.ID, 1, "DONE")' "$SUB"; then
  echo "PASS Submit marks stage-1 DONE (下单即完成环节1)"
else
  fail_step "Submit stage-1 DONE"
fi

# 2) 回填脚本存在 + 幂等标记(BEGIN/COMMIT 单事务 + finished_at IS NULL 守卫)。
BK="$ROOT/scripts/ops/stage-finished-at-backfill.sql"
if [ -f "$BK" ] && grep -q 'finished_at IS NULL' "$BK" && grep -q '^BEGIN;' "$BK" && grep -q '^COMMIT;' "$BK"; then
  echo "PASS backfill script exists with idempotent marker (finished_at IS NULL guard)"
else
  fail_step "backfill script idempotent marker"
fi

# 3) 102 现网抽查:DONE 行 finished_at 非空比例(回填后应 ≥25%,基线 2.4%)。
DONE_TOT=$(sql102 <<SQL || echo ERR
SELECT count(*) FROM order_stages WHERE result='DONE';
SQL
)
DONE_FIN=$(sql102 <<SQL || echo ERR
SELECT count(*) FROM order_stages WHERE result='DONE' AND finished_at IS NOT NULL;
SQL
)
if [ "$DONE_TOT" = "ERR" ] || [ "$DONE_FIN" = "ERR" ]; then
  fail_step "102 SQL unreachable"
elif [ "$DONE_TOT" -gt 0 ] && python3 -c "
import sys
tot, fin = $DONE_TOT, $DONE_FIN
ratio = fin / tot
print(f'ratio={ratio:.3f} ({fin}/{tot})')
sys.exit(0 if ratio >= 0.25 else 1)
"; then
  echo "PASS 102 DONE finished_at ratio >= 0.25 (was 0.024 baseline)"
else
  fail_step "102 DONE finished_at ratio below 0.25"
fi

# 4) live probe:现网新建 acc_ 订单,环节1 必须已写 finished_at(证明新写入路径生效),
#    验证后按 acc_ 标记清理(造数不过夜)。
API="$BASE_URL/api/admin/v1"
SUF="sfat$$"
PROBE=0
CLEANED=0
RISK_WAS="unknown"
cleanup_probe() {
  if [ "$CLEANED" = "0" ]; then
    sql102 <<SQL >/dev/null 2>&1
DELETE FROM order_stages WHERE order_id IN (SELECT id FROM orders WHERE address_id IN (SELECT id FROM addresses WHERE name LIKE '验收地址-sfat%'));
DELETE FROM orders WHERE address_id IN (SELECT id FROM addresses WHERE name LIKE '验收地址-sfat%');
DELETE FROM addresses WHERE name LIKE '验收地址-sfat%';
SQL
    CLEANED=1
  fi
  if [ "$RISK_WAS" = "true" ]; then
    curl -sS -m 10 -X PUT "$API/params/risk.direct.enabled" -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d '{"value":"true"}' >/dev/null 2>&1 || true
  fi
}
if [ -z "$KEY" ]; then
  fail_step "live probe skipped: no ADMIN_API_KEY"
else
  trap cleanup_probe EXIT
  # 临时关停直营风控(与 mainchain-acceptance 同姿势),结束后恢复。
  if curl -sS -m 10 "$API/params/risk.direct.enabled" -H "X-API-Key: $KEY" 2>/dev/null | grep -q '"value":"true"'; then
    RISK_WAS=true
    curl -sS -m 10 -X PUT "$API/params/risk.direct.enabled" -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d '{"value":"false"}' >/dev/null 2>&1 || true
  else
    RISK_WAS=false
  fi
  ADDR=$(curl -sS -m 15 -X POST "$API/addresses" -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d "{\"label\":\"acc_sfat_$SUF\",\"name\":\"验收地址-sfat-$SUF\"}" 2>/dev/null)
  ADDR_ID=$(echo "$ADDR" | python3 -c "import json,sys;d=json.load(sys.stdin);print(d.get('data',d).get('id',''))" 2>/dev/null)
  if [ -n "$ADDR_ID" ]; then
    ORD=$(curl -sS -m 15 -X POST "$API/orders" -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d "{\"customerId\":214,\"offerId\":101,\"addressId\":$ADDR_ID,\"channelId\":102}" 2>/dev/null)
    ORD_NO=$(echo "$ORD" | python3 -c "import json,sys;d=json.load(sys.stdin);print(d.get('data',d).get('orderNo',''))" 2>/dev/null)
    if [ -n "$ORD_NO" ]; then
      FIN=$(sql102 <<SQL || echo ERR
SELECT count(*) FROM order_stages os JOIN orders o ON o.id=os.order_id
WHERE o.order_no='$ORD_NO' AND os.stage=1 AND os.finished_at IS NOT NULL;
SQL
      )
      if [ "$FIN" = "1" ]; then
        echo "PASS live probe: new order $ORD_NO stage-1 finished_at written (新推进路径生效)"
        PROBE=1
      else
        fail_step "live probe: stage-1 finished_at missing for $ORD_NO (旧二进制?未部署?)"
      fi
    else
      fail_step "live probe: order create failed"
    fi
  else
    fail_step "live probe: address create failed"
  fi
  cleanup_probe
  trap - EXIT
fi
[ "$PROBE" = "1" ] || fail_step "live probe not satisfied"

if [ "$fail" -ne 0 ]; then echo "FAIL stage-finished-at-selftest"; exit 1; fi
echo "PASS stage-finished-at-selftest"
