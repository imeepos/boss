#!/usr/bin/env bash
# verify-contractor-settlement-e2e.sh -- W1 承包商与工程结算端到端实测(102 真实部署,禁 mock)。
# 断言: P0 造数(acc_w1 前缀,设施 CLS00098/CLS00097 PLANNED) / P1 建施工类供应商
#   / P2 建项目挂承包商 / P3 录清单(数量x单价,金额后端计算) / P4 开工+竣工 ACCEPTED
#   / P5 负例(ACCEPTED 后改定额拒) / P6 发起结算(应付=SUM(amount),断言金额与状态)
#   / P7 确认结算 SETTLED→作废 VOIDED→重开新单 / P8 未指定承包商拒结算
#   / RES 自清理残留为零+孤儿巡检。
# 成功断言一律核信封 d.code==0(仅看 HTTP 200 会把业务错误当假绿);
# 错误断言核信封业务码(40900/42200)。信号: 收尾 [e2e-w1-settlement RESULT: ...] 可 grep。
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
TAG="acc_w1"
SUP="${TAG}-sup-$(date +%H%M%S)"
PROJ="${TAG}-P1-$(date +%H%M%S)"
PROJ2="${TAG}-P2-$(date +%H%M%S)"
FAC1="CLS00098"
FAC2="CLS00097"

sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }
sqlval() { sql < <(echo "$1") | tr -d "[:space:]"; }

FAIL_N=0; FAILED_IDS=""
ok() { echo "PASS: [$1] $2"; }
bad() { FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"; echo "FAIL: [$1] $2" >&2; echo "[e2e-w1-settlement] ASSERTION FAILED id=$1" >&2; }
assert_eq() { if [ "$2" = "$3" ]; then ok "$1" "$2 ($4)"; else bad "$1" "期望=$2 实际=$3 上下文=$4 body=${BODY:0:120}"; fi; }
# assert_ok 成功路径守卫:HTTP 200 之外必须信封 code==0,否则业务错误假绿。
assert_ok() { local C; C=$(jf "$BODY" "d['code']"); if [ "$C" = "0" ]; then ok "$1" "$2"; else bad "$1" "信封code=$C 上下文=$2 body=${BODY:0:120}"; fi; }
assert_code() { local C; C=$(jf "$BODY" "d['code']"); assert_eq "$1" "$2" "$C" "$3"; }

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
  echo "清尾: 造数自清理($TAG 前缀,幂等)" >&2
  sql <<SQL
DELETE FROM construction_settlements WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE '${TAG}-%');
DELETE FROM construction_items WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE '${TAG}-%');
DELETE FROM construction_projects WHERE proj_no LIKE '${TAG}-%';
DELETE FROM procurement_suppliers WHERE code LIKE '${TAG}-%';
DELETE FROM odn_facility WHERE code IN ('$FAC1','$FAC2') AND name LIKE '${TAG} %';
SELECT 'settlements=' || count(*) FROM construction_settlements WHERE project_no LIKE '${TAG}-%'
UNION ALL SELECT 'projects=' || count(*) FROM construction_projects WHERE proj_no LIKE '${TAG}-%'
UNION ALL SELECT 'suppliers=' || count(*) FROM procurement_suppliers WHERE code LIKE '${TAG}-%'
UNION ALL SELECT 'facilities=' || count(*) FROM odn_facility WHERE code IN ('$FAC1','$FAC2') AND name LIKE '${TAG} %';
SQL
}

cleanup_data >/dev/null 2>&1

# P0 造数:两个 PLANNED 设施(清单两行用;ON CONFLICT 幂等,仅动本任务命名行)。
sqlval "INSERT INTO odn_facility(code, kind, prv_code, city_prefix, grid_code, name, status, lifecycle_status) VALUES ('$FAC1','CLS','PHL001','MNL',1,'$TAG 设施','IN_USE','PLANNED') ON CONFLICT (code) DO UPDATE SET lifecycle_status='PLANNED', name='$TAG 设施' RETURNING code;" >/dev/null
sqlval "INSERT INTO odn_facility(code, kind, prv_code, city_prefix, grid_code, name, status, lifecycle_status) VALUES ('$FAC2','CLS','PHL001','MNL',1,'$TAG 设施2','IN_USE','PLANNED') ON CONFLICT (code) DO UPDATE SET lifecycle_status='PLANNED', name='$TAG 设施2' RETURNING code;" >/dev/null
LC=$(sqlval "SELECT count(*) FROM odn_facility WHERE code IN ('$FAC1','$FAC2') AND lifecycle_status='PLANNED' AND name LIKE '${TAG} %';")
assert_eq P0 "2" "$LC" "设施 $FAC1/$FAC2 备妥"

# P1 建施工类供应商(contractorType=CONSTRUCTION + 资质;信封核 code==0)。
req POST /procurement/suppliers "{\"code\":\"$SUP\",\"name\":\"$TAG 施工承包商\",\"legalEntityId\":1,\"contactName\":\"王工\",\"contactPhone\":\"13800000001\",\"contractorType\":\"CONSTRUCTION\",\"qualification\":\"通信工程施工总承包三级\"}"
assert_ok P1 "建施工类供应商"
SUP_ID=$(jf "$BODY" "d['data']['id']")
req GET "/procurement/suppliers"
CT=$(jf "$BODY" "[s for s in d['data']['items'] if s['id']==$SUP_ID][0]['contractorType']")
assert_eq P1 "CONSTRUCTION" "$CT" "供应商承建类型回读(信封 data.items)"

# P2 建项目并挂承包商(PID 取数后看门,防空 URL 假绿)。
req POST /odn/constructions "{\"projNo\":\"$PROJ\",\"name\":\"$TAG 施工单\",\"prvCode\":\"PHL001\",\"cityPrefix\":\"MNL\"}"
assert_ok P2 "建施工项目"
PID=$(sqlval "SELECT id FROM construction_projects WHERE proj_no='$PROJ';")
case "$PID" in ''|*[!0-9]*) bad P2 "项目id取空(PID=$PID)"; exit 1 ;; esac
ok "P2" "项目id=$PID"
req PUT "/odn/constructions/$PID/contractor" "{\"contractorId\":$SUP_ID}"
assert_ok P2 "指定承包商"
req GET "/odn/constructions/$PID"
assert_eq P2 "$TAG 施工承包商" "$(jf "$BODY" "d['data']['project']['contractorName']")" "承包商名称快照"

# P3 录清单两行:2x350=700 + 1.5x80=120,合计 820(生成列后端计算)。
req POST "/odn/constructions/$PID/items" "{\"facilityCode\":\"$FAC1\",\"quantity\":2,\"unitPrice\":350}"
assert_ok P3 "追加明细1($FAC1)"
req POST "/odn/constructions/$PID/items" "{\"facilityCode\":\"$FAC2\",\"quantity\":1.5,\"unitPrice\":80}"
assert_ok P3 "追加明细2($FAC2)"
AMT=$(sqlval "SELECT SUM(amount) FROM construction_items WHERE project_id=$PID;")
assert_eq P3 "820.00" "$AMT" "生成列金额=数量x单价(700+120)"

# P4 开工+竣工。
req POST "/odn/constructions/$PID/start"
assert_ok P4 "开工"
req POST "/odn/constructions/$PID/accept" "{\"note\":\"$TAG as-built\"}"
assert_ok P4 "竣工"
req GET "/odn/constructions/$PID"
assert_eq P4 "ACCEPTED" "$(jf "$BODY" "d['data']['project']['status']")" "项目状态"

# P5 负例:ACCEPTED 后改定额被拒(明细锁定,40900)。
IID=$(sqlval "SELECT id FROM construction_items WHERE project_id=$PID ORDER BY id LIMIT 1;")
req PUT "/odn/constructions/$PID/items/$IID" "{\"quantity\":9,\"unitPrice\":9}"
assert_code P5 "40900" "ACCEPTED 后改定额拒(锁定)"

# P6 发起结算:应付=820,状态 PENDING;重复发起拒。
req POST "/odn/constructions/$PID/settlements"
assert_ok P6 "发起结算"
assert_eq P6 "820" "$(jf "$BODY" "d['data']['totalAmount']")" "应付金额=清单汇总"
assert_eq P6 "PENDING" "$(jf "$BODY" "d['data']['status']")" "结算单状态"
SID=$(jf "$BODY" "d['data']['id']")
req POST "/odn/constructions/$PID/settlements"
assert_code P6 "40900" "重复发起拒(同项目单有效)"

# P7 确认结算 SETTLED → 作废 VOIDED(原因必填) → 重开新单(金额不变)。
req POST "/odn/settlements/$SID/settle"
assert_ok P7 "确认结算"
req GET "/odn/settlements/$SID"
assert_eq P7 "SETTLED" "$(jf "$BODY" "d['data']['status']")" "已结算"
req POST "/odn/settlements/$SID/void" "{\"reason\":\"$TAG 纠错作废\"}"
assert_ok P7 "作废"
req POST "/odn/constructions/$PID/settlements"
assert_ok P7 "作废后重开(新单)"
assert_eq P7 "820" "$(jf "$BODY" "d['data']['totalAmount']")" "新单应付金额不变"
SID2=$(sqlval "SELECT id FROM construction_settlements WHERE project_id=$PID AND status='PENDING' ORDER BY id DESC LIMIT 1;")
req POST "/odn/settlements/$SID2/void" "{}"
assert_code P7 "42200" "作废缺原因拒"

# P8 未指定承包商的 ACCEPTED 项目拒结算(存量兼容口径,40900)。
req POST /odn/constructions "{\"projNo\":\"$PROJ2\",\"name\":\"$TAG 无承包商\"}"
assert_ok P8 "建无承包商项目"
PID2=$(sqlval "SELECT id FROM construction_projects WHERE proj_no='$PROJ2';")
req POST "/odn/constructions/$PID2/start"
req POST "/odn/constructions/$PID2/accept" "{\"note\":\"\"}"
req POST "/odn/constructions/$PID2/settlements"
assert_code P8 "40900" "未挂承包商拒结算"

cleanup_data
ORPHAN=$(sqlval "SELECT count(*) FROM construction_items WHERE project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE '${TAG}-%');")
assert_eq RES "0" "$ORPHAN" "孤儿巡检(结算单/项目/明细/供应商/设施归零)"

if [ "$FAIL_N" -eq 0 ]; then echo "[e2e-w1-settlement RESULT: PASS]"; else echo "[e2e-w1-settlement RESULT: FAIL] failed=$FAIL_N ids=$FAILED_IDS"; exit 1; fi