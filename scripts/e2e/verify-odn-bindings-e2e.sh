#!/usr/bin/env bash
# verify-odn-bindings-e2e.sh -- ODN P3 逻辑-物理绑定端到端实测(102 真实部署,禁 mock)。
# 断言: P0 造数(acc_obd 设备 OLT001+3 口,bindings 表缺失=等部署) / P1 分配 port1
#   / P2 开工 IN_SERVICE / P3 绑定订单 / P4 按口反查 / P5 按单反查 / P6 重复绑定被拒
#   / P7 非在网端口绑定被拒 / P8 解绑 / RES 自清理残留为零。
# 信号: PASS/FAIL 逐条;收尾 [e2e-odn-bindings RESULT: ...] 可 grep。
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
DEV="OLT001"

sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }
sqlval() { sql < <(echo "$1") | tr -d "[:space:]"; }

FAIL_N=0; FAILED_IDS=""
ok() { echo "PASS: [$1] $2"; }
bad() { FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"; echo "FAIL: [$1] $2" >&2; echo "[e2e-odn-bindings] ASSERTION FAILED id=$1" >&2; }
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
  echo "清尾: 造数自清理(幂等)" >&2
  sql <<SQL
DELETE FROM odn_bindings WHERE port_id IN (SELECT id FROM odn_port WHERE device_id IN (SELECT id FROM odn_device WHERE code='OLT001'));
DELETE FROM odn_port WHERE device_id IN (SELECT id FROM odn_device WHERE code='OLT001');
DELETE FROM odn_device WHERE code='OLT001';
SELECT 'bindings=' || count(*) FROM odn_bindings WHERE port_id NOT IN (SELECT id FROM odn_port)
UNION ALL SELECT 'ports=' || count(*) FROM odn_port WHERE device_id NOT IN (SELECT id FROM odn_device)
UNION ALL SELECT 'device=' || count(*) FROM odn_device WHERE code='OLT001';
SQL
}

seed_data() {
  cleanup_data >/dev/null 2>&1
  local has
  has=$(sqlval "SELECT to_regclass('public.odn_bindings') IS NOT NULL AND to_regclass('public.odn_port') IS NOT NULL;")
  if [ "$has" != "t" ]; then FAIL_REASON=deploy_pending; return 1; fi
  local city
  city=$(sqlval "SELECT prv_code || ',' || city_prefix FROM odn_city_code WHERE city_prefix='MNL' LIMIT 1;")
  [ -z "$city" ] && city=$(sqlval "SELECT prv_code || ',' || city_prefix FROM odn_city_code ORDER BY prv_code, city_prefix LIMIT 1;")
  if [ -z "$city" ]; then FAIL_REASON=no_city; return 1; fi
  local prv="${city%%,*}" cpre="${city##*,}"
  local devid
  sql <<SQL
DELETE FROM odn_port WHERE device_id IN (SELECT id FROM odn_device WHERE code='$DEV');
DELETE FROM odn_device WHERE code='$DEV';
SQL
  devid=$(sqlval "INSERT INTO odn_device(code, kind, prv_code, city_prefix) VALUES ('$DEV','OLT','$prv','$cpre') RETURNING id;")
  [ -z "$devid" ] && { FAIL_REASON=seed_device; return 1; }
  sql <<SQL
INSERT INTO odn_port(device_id, port_no) VALUES ($devid,1),($devid,2),($devid,3);
SQL
  echo "$devid"
}

echo "ODN P3 逻辑-物理绑定端到端实测 @ $BASE_URL"
DEVID=$(seed_data) || { bad "P0" "造数失败: ${FAIL_REASON:-}"; cleanup_data >/dev/null 2>&1; exit 1; }
ok "P0" "造数基线 设备=$DEV id=$DEVID 3 空闲口"

# P1 分配 port1 → RESERVED
req POST "/odn/devices/$DEVID/ports/allocate" "{\"orderId\":900}"
assert_eq "P1.code" "0" "$(jf "$BODY" "d['code']")" "分配"
req GET "/odn/devices/$DEVID/ports"
P1ID=$(jf "$BODY" "[x['id'] for x in d['data'] if x['portNo']==1][0]")
# P2 开工 → IN_SERVICE
req POST "/odn/ports/$P1ID/activate"
assert_eq "P2.activate" "IN_SERVICE" "$(jf "$BODY" "d['data']['status']")" "端口在网"

# P3 绑定订单
req POST "/odn/bindings" "{\"portId\":$P1ID,\"orderId\":900,\"note\":\"acc_obd\"}"
[ "${HTTP_CODE:-}" = "200" ] && assert_eq "P3.code" "0" "$(jf "$BODY" "d['code']")" "绑定" || bad "P3" "HTTP=${HTTP_CODE:-} body=${BODY:-}"

# P4 按口反查
req GET "/odn/bindings?portId=$P1ID"
assert_eq "P4.count" "1" "$(jf "$BODY" "len(d['data'])")" "绑定数"
assert_eq "P4.order" "900" "$(jf "$BODY" "d['data'][0]['orderId']")" "订单"

# P5 按单反查
req GET "/odn/bindings?orderId=900"
assert_eq "P5.count" "1" "$(jf "$BODY" "len(d['data'])")" "绑定数"

# P6 重复绑定被拒
req POST "/odn/bindings" "{\"portId\":$P1ID,\"orderId\":901}"
C=$(jf "$BODY" "d['code']")
if [ "$C" != "0" ]; then ok "P6" "重复绑定被拒 code=$C"; else bad "P6" "未被拒绝 body=$BODY"; fi

# P7 非在网端口绑定被拒(port2)
req GET "/odn/devices/$DEVID/ports"
P2ID=$(jf "$BODY" "[x['id'] for x in d['data'] if x['portNo']==2][0]")
req POST "/odn/bindings" "{\"portId\":$P2ID,\"orderId\":902}"
C=$(jf "$BODY" "d['code']")
if [ "$C" != "0" ]; then ok "P7" "非在网绑定被拒 code=$C"; else bad "P7" "未被拒绝 body=$BODY"; fi

# P8 解绑
req DELETE "/odn/bindings/port/$P1ID"
req GET "/odn/bindings?portId=$P1ID"
assert_eq "P8.count" "0" "$(jf "$BODY" "len(d['data'])")" "解绑后为空"

# RES 收尾
RES=$(cleanup_data 2>/dev/null | tr '\n' ' ')
echo "RES: $RES"
assert_eq "RES.bindings" "bindings=0" "$(echo "$RES" | grep -o 'bindings=[0-9]*' | head -1)" "绑定清理"
assert_eq "RES.ports" "ports=0" "$(echo "$RES" | grep -o 'ports=[0-9]*' | head -1)" "端口清理"
assert_eq "RES.device" "device=0" "$(echo "$RES" | grep -o 'device=[0-9]*' | head -1)" "设备清理"

if [ "$FAIL_N" -eq 0 ]; then echo "[e2e-odn-bindings RESULT: PASS]"; else echo "[e2e-odn-bindings RESULT: FAIL ids:$FAILED_IDS]"; exit 1; fi