#!/usr/bin/env bash
# verify-odn-impact-e2e.sh -- ODN P7 影响面查询端到端实测(102 真实部署,禁 mock)。
# 断言: P0 造数(acc_oim 前缀:设施+SERVED 覆盖+地址挂接客户) / P1 报告 code=0 含覆盖
#   / P2 客户清单命中 / P3 未知设施报错 / RES 自清理残留为零。
# 信号: PASS/FAIL 逐条;收尾 [e2e-odn-impact RESULT: ...] 可 grep。
# 依赖: curl python3 ssh(102 免密);404 = 新路由未随部署生效,等部署窗口重跑。
set -u

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
source "$ROOT/scripts/ops/acceptance-lock.sh"
acquire_acceptance_lock || exit 1
trap release_acceptance_lock EXIT

env_or() { local v; v=$(printenv "$1" 2>/dev/null); if [ -n "$v" ]; then echo "$v"; else echo "$2"; fi; }

ARG1=""; if [ $# -ge 1 ]; then ARG1="$1"; fi
BASE_URL=$(env_or BASE_URL "http://192.168.0.102:28080")
case "$ARG1" in http*) BASE_URL="$ARG1" ;; esac
API="$BASE_URL/api/admin/v1"
KEY=$(env_or ADMIN_API_KEY "$(python3 -c "import json;print(json.load(open('$ROOT/.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key'])")")
SSH_HOST=$(env_or SSH_HOST "imeepos@192.168.0.102")
FAC="CLS00097"
MARK="acc_oim"

sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }
sqlval() { sql < <(echo "$1") | tr -d "[:space:]"; }

FAIL_N=0; FAILED_IDS=""
ok() { echo "PASS: [$1] $2"; }
bad() { FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"; echo "FAIL: [$1] $2" >&2; echo "[e2e-odn-impact] ASSERTION FAILED id=$1" >&2; }
assert_eq() { if [ "$2" = "$3" ]; then ok "$1" "$2 ($4)"; else bad "$1" "期望=$2 实际=$3 上下文=$4 body=${BODY:0:100}"; fi; }

req() {
  local method="$1" path="$2" body="${3:-}" out
  out=$(curl -sS -m 30 -w $'\n%{http_code}' -X "$method" "$API$path" -H "X-API-Key: $KEY" \
    -H "Content-Type: application/json" ${body:+-d "$body"} 2>/dev/null) || { HTTP_CODE="000"; BODY="curl-fail"; return 1; }
  HTTP_CODE=$(printf '%s\n' "$out" | tail -n1 | tr -d "[:space:]")
  BODY=$(printf '%s\n' "$out" | sed '$d')
}

jf() { python3 -c '
import json, sys
try:
    d = json.loads(sys.argv[1])
    print(eval(sys.argv[2]))
except Exception as e:
    print("<JF-FAIL " + repr(e)[:70] + ">")
' "$1" "$2"; }

cleanup_data() {
  echo "清尾: 造数自清理($MARK 痕迹,幂等)" >&2
  sql <<SQL
DELETE FROM customers WHERE name LIKE '$MARK%';
DELETE FROM address_coverage WHERE facility_code='$FAC' OR note LIKE '$MARK%';
DELETE FROM odn_facility WHERE code='$FAC';
SELECT 'customers=' || count(*) FROM customers WHERE name LIKE '$MARK%'
UNION ALL SELECT 'coverage=' || count(*) FROM address_coverage WHERE note LIKE '$MARK%'
UNION ALL SELECT 'facility=' || count(*) FROM odn_facility WHERE code='$FAC';
SQL
}

ensure_table() {
  local has
  has=$(sqlval "SELECT to_regclass('public.address_coverage') IS NOT NULL;")
  if [ "$has" = "t" ]; then return 0; fi
  if [ ! -f "$ROOT/migrations/000197_address_coverage.up.sql" ]; then echo "迁移文件不存在" >&2; return 1; fi
  sql < "$ROOT/migrations/000197_address_coverage.up.sql" >/dev/null || return 1
  sqlval "SELECT to_regclass('public.address_coverage') IS NOT NULL;"
}

seed_data() {
  cleanup_data >/dev/null 2>&1
  ensure_table >/dev/null || { FAIL_REASON=ensure_table; return 1; }
  local city addr ent
  city=$(sqlval "SELECT prv_code || ',' || city_prefix FROM odn_city_code WHERE city_prefix='MNL' LIMIT 1;")
  [ -z "$city" ] && city=$(sqlval "SELECT prv_code || ',' || city_prefix FROM odn_city_code ORDER BY prv_code, city_prefix LIMIT 1;")
  addr=$(sqlval "SELECT min(id) FROM addresses;")
  ent=$(sqlval "SELECT min(id) FROM legal_entities;")
  if [ -z "$city" ] || [ -z "$addr" ] || [ -z "$ent" ]; then FAIL_REASON=no_seed; return 1; fi
  local prv="${city%%,*}" cpre="${city##*,}"
  sqlval "INSERT INTO odn_facility(code, kind, prv_code, city_prefix, name) VALUES ('$FAC','CLS','$prv','$cpre','$MARK 设施') RETURNING code;" >/dev/null 2>&1 || { FAIL_REASON=seed_facility; return 1; }
  sqlval "INSERT INTO customers(legal_entity_id, address_id, region_id, region_name, name, phone, id_type) VALUES ($ent, $addr, 0, 'e2e', '$MARK 客户', '13900000000', '无') RETURNING id;" >/dev/null 2>&1 || { FAIL_REASON=seed_customer; return 1; }
  echo "$addr"
}

echo "ODN P7 影响面查询端到端实测 @ $BASE_URL"
ADDR=$(seed_data) || { bad "P0" "造数失败: ${FAIL_REASON:-}"; cleanup_data >/dev/null 2>&1; exit 1; }
ok "P0" "造数基线 设施=$FAC 地址=$ADDR 客户=$MARK"

# P0b 经 API 建覆盖关联(顺带验证 coverage 接口)
req POST "/odn/coverage" "{\"addressId\":$ADDR,\"facilityCode\":\"$FAC\",\"status\":\"SERVED\",\"note\":\"$MARK\"}"
assert_eq "P0b.code" "0" "$(jf "$BODY" "d['code']")" "覆盖关联建立"

# P1 影响面报告
req GET "/odn/impact?facilityCode=$FAC"
[ "${HTTP_CODE:-}" = "200" ] || bad "P1.http" "期望=200 实际=${HTTP_CODE:-}"
assert_eq "P1.code" "0" "$(jf "$BODY" "d['code']")" "业务码"
assert_eq "P1.facility" "$FAC" "$(jf "$BODY" "d['data']['facilityCode']")" "设施摘要"
assert_eq "P1.covCount" "1" "$(jf "$BODY" "len(d['data']['coverages'])")" "受影响覆盖数"
assert_eq "P1.covStatus" "SERVED" "$(jf "$BODY" "d['data']['coverages'][0]['status']")" "覆盖状态"

# P2 客户清单命中(种子 min(id) 地址与真实客户共存:断言造数客户被包含,2026-09-07 修)
assert_eq "P2.custContains" "True" "$(jf "$BODY" "str(any(x['name']=='$MARK 客户' for x in d['data']['customers']))")" "造数客户命中"
assert_eq "P2.custCount" "True" "$(jf "$BODY" "str(len(d['data']['customers'])>=1)")" "客户数≥1"

# P3 未知设施报错
req GET "/odn/impact?facilityCode=CLS99999"
C=$(jf "$BODY" "d['code']")
if [ "$C" != "0" ]; then ok "P3" "未知设施报错 code=$C"; else bad "P3" "未知设施未报错 body=$BODY"; fi

# RES 收尾
RES=$(cleanup_data 2>/dev/null | tr '\n' ' ')
echo "RES: $RES"
assert_eq "RES.customers" "customers=0" "$(echo "$RES" | grep -o 'customers=[0-9]*' | head -1)" "客户清理"
assert_eq "RES.coverage" "coverage=0" "$(echo "$RES" | grep -o 'coverage=[0-9]*' | head -1)" "覆盖清理"
assert_eq "RES.facility" "facility=0" "$(echo "$RES" | grep -o 'facility=[0-9]*' | head -1)" "设施清理"

if [ "$FAIL_N" -eq 0 ]; then echo "[e2e-odn-impact RESULT: PASS]"; else echo "[e2e-odn-impact RESULT: FAIL ids:$FAILED_IDS]"; exit 1; fi