#!/usr/bin/env bash
# verify-odn-port-e2e.sh -- ODN P2 物理端口占用态端到端实测(102 真实部署,禁 mock)。
# 断言: P0 造数(acc_opn 设备 OLT001+3 空闲口,表缺失补 apply 000200) / P1 分配 port1
#   / P2 再分配 port2 / P3 释放 port1 / P4 按地址分配命中 port1 / P5 开工 IN_SERVICE
#   / P6 拆机释放 / P7 分配耗尽报错 / RES 自清理残留为零。
# 信号: PASS/FAIL 逐条;收尾 [e2e-odn-port RESULT: ...] 可 grep。
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
DEV="OLT001"
MARK="acc_opn"

sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }
sqlval() { sql < <(echo "$1") | tr -d "[:space:]"; }

FAIL_N=0; FAILED_IDS=""
ok() { echo "PASS: [$1] $2"; }
bad() { FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"; echo "FAIL: [$1] $2" >&2; echo "[e2e-odn-port] ASSERTION FAILED id=$1" >&2; }
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
  echo "清尾: 造数自清理($DEV,幂等)" >&2
  sql <<SQL
DELETE FROM address_coverage WHERE device_id IN (SELECT id FROM odn_device WHERE code='OLT001') OR note LIKE 'acc_opn%';
DELETE FROM odn_port WHERE device_id IN (SELECT id FROM odn_device WHERE code='OLT001');
DELETE FROM odn_device WHERE code='OLT001';
SELECT 'ports=' || count(*) FROM odn_port WHERE device_id NOT IN (SELECT id FROM odn_device)
UNION ALL SELECT 'device=' || count(*) FROM odn_device WHERE code='OLT001';
SQL
}

ensure_table() {
  local has
  has=$(sqlval "SELECT to_regclass('public.odn_port') IS NOT NULL;")
  if [ "$has" = "t" ]; then return 0; fi
  echo "odn_port 缺表,等待部署管道迁移(不手工 apply,防 schema_migrations 漏登记崩溃循环)" >&2
  return 1
}

seed_data() {
  ensure_table || { FAIL_REASON=deploy_pending; return 1; }
  cleanup_data >/dev/null 2>&1
  local city
  city=$(sqlval "SELECT prv_code || ',' || city_prefix FROM odn_city_code WHERE city_prefix='MNL' LIMIT 1;")
  [ -z "$city" ] && city=$(sqlval "SELECT prv_code || ',' || city_prefix FROM odn_city_code ORDER BY prv_code, city_prefix LIMIT 1;")
  if [ -z "$city" ]; then FAIL_REASON=no_city; return 1; fi
  local prv="${city%%,*}" cpre="${city##*,}"
  local devid
  sql <<SQL
DELETE FROM address_coverage WHERE device_id IN (SELECT id FROM odn_device WHERE code='$DEV') OR note LIKE '$MARK%';
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

echo "ODN P2 物理端口占用态端到端实测 @ $BASE_URL"
DEVID=$(seed_data) || { bad "P0" "造数失败: ${FAIL_REASON:-}(deploy_pending=等部署窗口迁移)"; cleanup_data >/dev/null 2>&1; exit 1; }
ok "P0" "造数基线 设备=$DEV id=$DEVID 3 空闲口"

# P1 首次分配 → port 1
req POST "/odn/devices/$DEVID/ports/allocate" "{\"orderId\":777}"
[ "${HTTP_CODE:-}" = "200" ] && assert_eq "P1.port" "1" "$(jf "$BODY" "d['data']['portNo']")" "首次分配" || bad "P1" "HTTP=${HTTP_CODE:-} body=${BODY:-}"

# P2 再分配 → port 2
req POST "/odn/devices/$DEVID/ports/allocate" "{\"orderId\":778}"
assert_eq "P2.port" "2" "$(jf "$BODY" "d['data']['portNo']")" "顺序分配"

# P3 释放 port1
req GET "/odn/devices/$DEVID/ports"
P1ID=$(jf "$BODY" "[x['id'] for x in d['data'] if x['portNo']==1][0]")
req POST "/odn/ports/$P1ID/release"
[ "${HTTP_CODE:-}" = "200" ] && ok "P3" "port1 释放" || bad "P3" "HTTP=${HTTP_CODE:-} body=${BODY:-}"

# P4 按地址分配(覆盖关联兑现) → 复用 port1
# 前置:地址 288 覆盖挂接本设备(SERVED),allocate-for-address 才有判据
req POST "/odn/coverage" "{\"addressId\":288,\"deviceId\":$DEVID,\"status\":\"SERVED\",\"note\":\"acc_opn\"}"
req POST "/odn/ports/allocate-for-address" "{\"addressId\":288,\"orderId\":888}"
assert_eq "P4.port" "1" "$(jf "$BODY" "d['data']['portNo']")" "按地址分配命中已释放口"

# P5 开工 → IN_SERVICE
req GET "/odn/devices/$DEVID/ports"
P1ID=$(jf "$BODY" "[x['id'] for x in d['data'] if x['portNo']==1][0]")
req POST "/odn/ports/$P1ID/activate"
req GET "/odn/devices/$DEVID/ports"
assert_eq "P5.status" "IN_SERVICE" "$(jf "$BODY" "[x['status'] for x in d['data'] if x['portNo']==1][0]")" "开通在网"

# P6 拆机释放
req POST "/odn/ports/$P1ID/release"
req GET "/odn/devices/$DEVID/ports"
assert_eq "P6.status" "IDLE" "$(jf "$BODY" "[x['status'] for x in d['data'] if x['portNo']==1][0]")" "拆机释放"

# P7 分配耗尽报错(3 口全占:779→1, 780→3, 781 无空闲)
req POST "/odn/devices/$DEVID/ports/allocate" "{\"orderId\":779}"
req POST "/odn/devices/$DEVID/ports/allocate" "{\"orderId\":780}"
req POST "/odn/devices/$DEVID/ports/allocate" "{\"orderId\":781}"
C=$(jf "$BODY" "d['code']")
if [ "$C" != "0" ]; then ok "P7" "耗尽报错 code=$C"; else bad "P7" "端口耗尽未报错 body=$BODY"; fi

# RES 收尾
RES=$(cleanup_data 2>/dev/null | tr '\n' ' ')
echo "RES: $RES"
assert_eq "RES.ports" "ports=0" "$(echo "$RES" | grep -o 'ports=[0-9]*' | head -1)" "端口清理"
assert_eq "RES.device" "device=0" "$(echo "$RES" | grep -o 'device=[0-9]*' | head -1)" "设备清理"

if [ "$FAIL_N" -eq 0 ]; then echo "[e2e-odn-port RESULT: PASS]"; else echo "[e2e-odn-port RESULT: FAIL ids:$FAILED_IDS]"; exit 1; fi