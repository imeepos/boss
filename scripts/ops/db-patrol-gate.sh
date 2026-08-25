#!/usr/bin/env bash
# db-patrol-gate: 软引用孤儿巡检门禁(2026-08-25 审计 §五.2 落地,2026-08-29 固化)。
# 用途:
#   1) CI/deploy 后置门禁:任一巡检类孤儿数 > 阈值 → 退出 1,拦截 E2E 造数泄漏进 main;
#   2) 102 定时巡检(cron):每日跑一次,日志落 /var/log/boss-patrol-gate.log;
#   3) 主链路验收脚本收尾自检(mainchain-acceptance.sh 末尾调用)。
# 用法: scripts/ops/db-patrol-gate.sh [--max N]   # --max 各类允许的孤儿上限,默认 0
# 环境: BASE_URL / ADMIN_API_KEY 可覆盖;缺省 102 + test-accounts.json admin key。
# 自测: --selftest 用内置样例响应(含孤儿)验证解析与退出码,不访问网络。
set -u

MAX=0
SELFTEST=0
case "${1:-}" in
  --max) MAX="${2:-0}" ;;
  --selftest) SELFTEST=1 ;;
esac

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BASE_URL="${BASE_URL:-http://192.168.0.102:28080}"
API_KEY="${ADMIN_API_KEY:-$(python3 -c "import json;print(json.load(open('$ROOT/.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key'])" 2>/dev/null || true)}"

# judge: 读 stdin JSON 巡检响应,打印各类计数;任一超阈值 → exit 1。
judge() {
  MAX_ORPHANS="$MAX" python3 -c '
import json, os, sys
max_orphans = int(os.environ["MAX_ORPHANS"])
body = json.load(sys.stdin)
items = body.get("data", {}).get("items", [])
bad = []
for it in items:
    n = it.get("orphans", 0)
    check = it.get("check", "?")
    print(f"  {check}: {n}")
    if n > max_orphans:
        bad.append((check, n, it.get("sampleIds", [])[:5]))
if bad:
    print(f"ORPHAN-GATE FAIL: {len(bad)} check(s) over limit {max_orphans}:")
    for check, n, ids in bad:
        print(f"  - {check}: {n} orphans, sample {ids}")
    sys.exit(1)
print(f"ORPHAN-GATE OK: {len(items)} checks, all <= {max_orphans}")
'
}

if [ "$SELFTEST" = 1 ]; then
  echo "[selftest] leak payload (140 orphan orders) must be rejected:"
  if echo '{"code":0,"data":{"items":[{"check":"orders.customer_id -> customers","orphans":140,"sampleIds":[402]},{"check":"orders.offer_id -> product_offers","orphans":0,"sampleIds":[]}]}}' | judge; then
    echo "[selftest] FAIL: leak payload not detected"; exit 1
  fi
  echo "[selftest] clean payload must pass:"
  echo '{"code":0,"data":{"items":[{"check":"orders.customer_id -> customers","orphans":0,"sampleIds":[]}]}}' | judge || { echo "[selftest] FAIL"; exit 1; }
  echo "[selftest] PASS"
  exit 0
fi

if [ -z "$API_KEY" ]; then
  echo "ORPHAN-GATE FAIL: no ADMIN_API_KEY and test-accounts.json unavailable" >&2
  exit 2
fi
body=$(curl -sS -m 30 -H "X-API-Key: $API_KEY" "$BASE_URL/api/admin/v1/db-patrol/orphans")
if [ -z "$body" ]; then
  echo "ORPHAN-GATE FAIL: empty response from $BASE_URL" >&2
  exit 2
fi
echo "$body" | judge
