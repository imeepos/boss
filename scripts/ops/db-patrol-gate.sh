#!/usr/bin/env bash
# db-patrol-gate: 软引用孤儿巡检门禁(2026-08-25 审计 §五.2 落地,2026-08-29 固化)。
# 用途:
#   1) CI/deploy 后置门禁:任一巡检类孤儿数 > 阈值 → 退出 1,拦截 E2E 造数泄漏进 main;
#   2) 102 定时巡检(cron):每日跑一次,日志落 /var/log/boss-patrol-gate.log;
#   3) 主链路验收脚本收尾自检(mainchain-acceptance.sh 末尾调用)。
# 用法: scripts/ops/db-patrol-gate.sh [--max N]   # --max 各类允许的孤儿上限,默认 0
#        scripts/ops/db-patrol-gate.sh --asset    # 只跑资产两查(P4-T3),不跑孤儿门禁
# 环境: BASE_URL / ADMIN_API_KEY 可覆盖;缺省 102 + test-accounts.json admin key。
#       SSH_HOST 缺省 imeepos@192.168.0.102(开发机/CI 走 ssh);置空或 ssh 不可达
#       (如 102 本机 cron 自连无免密)自动回退本机 docker exec,两形态输出一致。
# 自测: --selftest 用内置样例响应(含孤儿)验证解析与退出码,不访问网络。
# 资产两查(P4-T3,只暴露不修改,不影响退出码):
#   ① [db-patrol] ASSET-EPC-INVALID count=N + SAMPLE 前 20 行——tags.epc_code 非 24-hex,
#     口径=贴标待回填/待清理(物理 EPC 与实物一致,严禁程序重写);
#   ② [db-patrol] ASSET-TYPE-UNKNOWN count=N + BREAKDOWN——assets.type 非白名单计数防新方言,
#     白名单 ONU/ROUTER/OLT 与 internal/domain/asset/type_whitelist.go 同步维护。
set -u

MAX=0
SELFTEST=0
ASSET_ONLY=0
case "${1:-}" in
  --max) MAX="${2:-0}" ;;
  --selftest) SELFTEST=1 ;;
  --asset) ASSET_ONLY=1 ;;
esac

SSH_HOST="${SSH_HOST:-imeepos@192.168.0.102}"
PG_CONTAINER="${PG_CONTAINER:-boss-infra-postgres-1}"
PG_USER="${PG_USER:-boss}"
PG_DB="${PG_DB:-boss}"

# psql_run: SQL 走 stdin。缺省经 ssh 到 102 容器;ssh 失败(含 102 cron 自连被拒)
# 回退本机 docker exec(102 上 imeepos 具备 docker 权限,已实测)。
psql_run() {
  if [ -n "$SSH_HOST" ]; then
    if out=$(ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" \
        "docker exec -i $PG_CONTAINER psql -U $PG_USER -d $PG_DB -v ON_ERROR_STOP=1 -q -tA" 2>/dev/null); then
      printf '%s' "$out"
      return 0
    fi
    echo "[db-patrol] ALERT sql via local docker exec (ssh $SSH_HOST unavailable)" >&2
  fi
  docker exec -i "$PG_CONTAINER" psql -U "$PG_USER" -d "$PG_DB" -v ON_ERROR_STOP=1 -q -tA
}

# asset_patrol: 资产两查(P4-T3)。前缀固定可 grep;只暴露不修改,恒 exit 0。
asset_patrol() {
  psql_run <<'SQL'
SELECT '[db-patrol] ASSET-EPC-INVALID count=' || count(*) FROM tags WHERE COALESCE(epc_code, '') !~ '^[0-9A-Fa-f]{24}$';
SELECT '[db-patrol] ASSET-EPC-INVALID-SAMPLE tag_id=' || id || ' tag_no=' || tag_no || ' epc=' || epc_code
  FROM tags WHERE COALESCE(epc_code, '') !~ '^[0-9A-Fa-f]{24}$' ORDER BY id LIMIT 20;
SELECT '[db-patrol] ASSET-TYPE-UNKNOWN count=' || COALESCE(sum(n), 0) FROM (
  SELECT count(*) AS n FROM assets WHERE COALESCE(type, '') NOT IN ('ONU', 'ROUTER', 'OLT')) s;
SELECT '[db-patrol] ASSET-TYPE-UNKNOWN-BREAKDOWN type=' || type || ' n=' || count(*)
  FROM assets WHERE COALESCE(type, '') NOT IN ('ONU', 'ROUTER', 'OLT')
  GROUP BY type ORDER BY count(*) DESC, type;
SQL
}

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

if [ "$ASSET_ONLY" = 1 ]; then
  asset_patrol
  exit 0
fi

if [ -z "$API_KEY" ]; then
  echo "ORPHAN-GATE FAIL: no ADMIN_API_KEY and test-accounts.json unavailable" >&2
  exit 2
fi
# 资产两查先跑(输出恒留痕,不受孤儿门禁退出码影响);门禁判定收尾,退出码=孤儿门禁。
asset_patrol
body=$(curl -sS -m 30 -H "X-API-Key: $API_KEY" "$BASE_URL/api/admin/v1/db-patrol/orphans")
if [ -z "$body" ]; then
  echo "ORPHAN-GATE FAIL: empty response from $BASE_URL" >&2
  exit 2
fi
echo "$body" | judge
