#!/usr/bin/env bash
# verify-odn-coverage-e2e.sh -- ODN P1 覆盖关联端到端实测(102 真实部署,禁 mock)。
# 断言: P0 造数(acc_ocv 前缀,幂等;表缺失补 apply 000197) / P1 POST SERVED / P2 回读
#   / P3 就近命中(≈22m SERVED) / P4 远点 UNSERVED / P5 列表包含 / P6 幂等改写 PENDING
#   / P7 SERVED 无目标被拒 / RES 自清理+残留为零;上报 odn_site 行数(残留核查)。
# 信号: PASS/FAIL 逐条;收尾 [e2e-odn-coverage RESULT: ...] 可 grep。
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
FAC_CODE="CLS00099"
MARK="acc_ocv"

sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }
sqlval() { sql < <(echo "$1") | tr -d "[:space:]"; }

FAIL_N=0; FAILED_IDS=""
ok() { echo "PASS: [$1] $2"; }
bad() { FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"; echo "FAIL: [$1] $2" >&2; echo "[e2e-odn-coverage] ASSERTION FAILED id=$1" >&2; }
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
DELETE FROM address_coverage WHERE facility_code='$FAC_CODE' OR note LIKE '$MARK%';
DELETE FROM odn_facility WHERE code='$FAC_CODE';
SELECT 'coverage=' || count(*) FROM address_coverage WHERE note LIKE '$MARK%'
UNION ALL SELECT 'facility=' || count(*) FROM odn_facility WHERE code='$FAC_CODE'
UNION ALL SELECT 'odn_site=' || count(*) FROM odn_site;
SQL
}

ensure_table() {
  local has
  has=$(sqlval "SELECT to_regclass('public.address_coverage') IS NOT NULL;")
  if [ "$has" = "t" ]; then return 0; fi
  echo "address_coverage 缺表,补 apply 000197(部署管道迁移前置)" >&2
  if [ ! -f "$ROOT/migrations/000197_address_coverage.up.sql" ]; then echo "迁移文件不存在" >&2; return 1; fi
  sql < "$ROOT/migrations/000197_address_coverage.up.sql" >/dev/null || return 1
  sqlval "SELECT to_regclass('public.address_coverage') IS NOT NULL;"
}

seed_data() {
  cleanup_data >/dev/null 2>&1
  ensure_table >/dev/null || { FAIL_REASON=ensure_table; return 1; }
  local city addr
  city=$(sqlval "SELECT prv_code || ',' || city_prefix FROM odn_city_code WHERE city_prefix='MNL' LIMIT 1;")
  [ -z "$city" ] && city=$(sqlval "SELECT prv_code || ',' || city_prefix FROM odn_city_code ORDER BY prv_code, city_prefix LIMIT 1;")
  addr=$(sqlval "SELECT min(id) FROM addresses;")
  if [ -z "$city" ] || [ -z "$addr" ]; then FAIL_REASON=no_seed; return 1; fi
  local prv="${city%%,*}" cpre="${city##*,}"
  sqlval "INSERT INTO odn_facility(code, kind, prv_code, city_prefix, name, lat, lng) VALUES ('$FAC_CODE','CLS','$prv','$cpre','$MARK 设施',14.5995,120.9842) RETURNING code;" >/dev/null || { FAIL_REASON=seed_facility; return 1; }
  echo "$addr"
}

echo "ODN P1 覆盖关联端到端实测 @ $BASE_URL"
ADDR=$(seed_data) || { bad "P0" "造数失败: ${FAIL_REASON:-}"; cleanup_data >/dev/null 2>&1; exit 1; }
ok "P0" "造数基线 $MARK 设施=$FAC_CODE 地址=$ADDR"

# P1 保存覆盖关联
req POST "/odn/coverage" "{\"addressId\":$ADDR,\"facilityCode\":\"$FAC_CODE\",\"status\":\"SERVED\",\"note\":\"$MARK\"}"
[ "${HTTP_CODE:-}" = "200" ] && assert_eq "P1.code" "0" "$(jf "$BODY" "d['code']")" "业务码" || bad "P1" "HTTP=${HTTP_CODE:-} body=${BODY:-}"

# P2 回读
req GET "/odn/coverage?addressId=$ADDR"
[ "${HTTP_CODE:-}" = "200" ] || bad "P2.http" "期望=200 实际=${HTTP_CODE:-}"
assert_eq "P2.status" "SERVED" "$(jf "$BODY" "d['data']['status']")" "可装状态回读"
assert_eq "P2.facility" "$FAC_CODE" "$(jf "$BODY" "d['data']['facilityCode']")" "设施回读"

# P3 就近命中(约 22m)
req GET "/odn/coverage/resolve?lat=14.5997&lng=120.9844"
assert_eq "P3.status" "SERVED" "$(jf "$BODY" "d['data']['status']")" "就近判定"
assert_eq "P3.facility" "$FAC_CODE" "$(jf "$BODY" "d['data']['facilityCode']")" "命中设施"
D=$(jf "$BODY" "round(d['data']['distanceM'])")
if [ "${D:-999999}" -lt 2000 ] 2>/dev/null; then ok "P3.distance" "距离 ${D}m < 2km"; else bad "P3.distance" "距离异常 D=$D"; fi

# P4 远点未覆盖
req GET "/odn/coverage/resolve?lat=19.0&lng=121.0"
assert_eq "P4.status" "UNSERVED" "$(jf "$BODY" "d['data']['status']")" "远点判定"

# P5 列表包含
req GET "/odn/coverage/list?limit=200"
assert_eq "P5.contains" "True" "$(jf "$BODY" "str(any(x['addressId']==$ADDR for x in d['data']))")" "列表含该地址"

# P6 幂等覆盖改写
req POST "/odn/coverage" "{\"addressId\":$ADDR,\"facilityCode\":\"$FAC_CODE\",\"status\":\"PENDING\",\"note\":\"$MARK\"}"
req GET "/odn/coverage?addressId=$ADDR"
assert_eq "P6.status" "PENDING" "$(jf "$BODY" "d['data']['status']")" "upsert 改写"

# P7 非法:SERVED 无目标
req POST "/odn/coverage" "{\"addressId\":$ADDR,\"status\":\"SERVED\"}"
C=$(jf "$BODY" "d['code']")
if [ "$C" != "0" ]; then ok "P7" "非法关联被拒 code=$C"; else bad "P7" "SERVED 无目标未被拒绝 body=$BODY"; fi

# RES 收尾
RES=$(cleanup_data 2>/dev/null | tr '\n' ' ')
echo "RES: $RES"
assert_eq "RES.coverage" "coverage=0" "$(echo "$RES" | grep -o 'coverage=[0-9]*' | head -1)" "覆盖造数清理"
assert_eq "RES.facility" "facility=0" "$(echo "$RES" | grep -o 'facility=[0-9]*' | head -1)" "设施造数清理"
SITE_N=$(echo "$RES" | grep -o 'odn_site=[0-9]*' | head -1 | cut -d= -f2)
echo "NOTE: odn_site 行数=$SITE_N(2026-09-06 对账存在 1 行验收残留,非本脚本造数,另行核查)"

if [ "$FAIL_N" -eq 0 ]; then echo "[e2e-odn-coverage RESULT: PASS]"; else echo "[e2e-odn-coverage RESULT: FAIL ids:$FAILED_IDS]"; exit 1; fi