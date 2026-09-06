#!/usr/bin/env bash
# verify-odn-construction-e2e.sh -- ODN P6 施工单端到端实测(102 真实部署,禁 mock)。
# 断言: P0 造数(acc_ocn 前缀,表缺失补 apply 000199,PLANNED 设施) / P1 建单 / P2 加明细
#   / P3 开工(设施 IN_BUILD) / P4 竣工(ACCEPTED+设施 IN_SERVICE+as-built 落) / P5 详情含明细
#   / P6 ACCEPTED 后加明细被拒 / P7 ACCEPTED 后再开工被拒 / RES 自清理残留为零。
# 信号: PASS/FAIL 逐条;收尾 [e2e-odn-construction RESULT: ...] 可 grep。
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
PROJ="acc_ocn-P1"
FAC="CLS00098"

sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }
sqlval() { sql < <(echo "$1") | tr -d "[:space:]"; }

FAIL_N=0; FAILED_IDS=""
ok() { echo "PASS: [$1] $2"; }
bad() { FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"; echo "FAIL: [$1] $2" >&2; echo "[e2e-odn-construction] ASSERTION FAILED id=$1" >&2; }
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
  echo "清尾: 造数自清理($PROJ/$FAC,幂等)" >&2
  sql <<SQL
DELETE FROM construction_items WHERE facility_code='$FAC' OR project_id IN (SELECT id FROM construction_projects WHERE proj_no LIKE 'acc_ocn-%');
DELETE FROM construction_projects WHERE proj_no LIKE 'acc_ocn-%';
DELETE FROM odn_facility WHERE code='$FAC';
SELECT 'projects=' || count(*) FROM construction_projects WHERE proj_no LIKE 'acc_ocn-%'
UNION ALL SELECT 'facility=' || count(*) FROM odn_facility WHERE code='$FAC';
SQL
}

ensure_table() {
  local has
  has=$(sqlval "SELECT to_regclass('public.construction_projects') IS NOT NULL;")
  if [ "$has" = "t" ]; then return 0; fi
  echo "construction_projects 缺表,补 apply 000199" >&2
  if [ ! -f "$ROOT/migrations/000199_odn_construction.up.sql" ]; then echo "迁移文件不存在" >&2; return 1; fi
  sql < "$ROOT/migrations/000199_odn_construction.up.sql" >/dev/null || return 1
  sqlval "SELECT to_regclass('public.construction_projects') IS NOT NULL;"
}

seed_data() {
  cleanup_data >/dev/null 2>&1
  ensure_table >/dev/null || { FAIL_REASON=ensure_table; return 1; }
  local city
  city=$(sqlval "SELECT prv_code || ',' || city_prefix FROM odn_city_code WHERE city_prefix='MNL' LIMIT 1;")
  [ -z "$city" ] && city=$(sqlval "SELECT prv_code || ',' || city_prefix FROM odn_city_code ORDER BY prv_code, city_prefix LIMIT 1;")
  if [ -z "$city" ]; then FAIL_REASON=no_city_seed; return 1; fi
  local prv="${city%%,*}" cpre="${city##*,}"
  sqlval "INSERT INTO odn_facility(code, kind, prv_code, city_prefix, name, lifecycle_status) VALUES ('$FAC','CLS','$prv','$cpre','acc_ocn 规划设施','PLANNED') RETURNING code;" >/dev/null 2>&1 || { FAIL_REASON=seed_facility; return 1; }
  echo "$prv $cpre"
}

echo "ODN P6 施工单端到端实测 @ $BASE_URL"
CITY=$(seed_data) || { bad "P0" "造数失败: ${FAIL_REASON:-}"; cleanup_data >/dev/null 2>&1; exit 1; }
ok "P0" "造数基线 设施=$FAC(PLANNED) 城市=[$CITY]"

# P1 建单
req POST "/odn/constructions" "{\"projNo\":\"$PROJ\",\"name\":\"acc_ocn 施工单\"}"
[ "${HTTP_CODE:-}" = "200" ] && assert_eq "P1.code" "0" "$(jf "$BODY" "d['code']")" "业务码" || bad "P1" "HTTP=${HTTP_CODE:-} body=${BODY:-}"

# 取单 id
req GET "/odn/constructions?limit=100"
PID=$(jf "$BODY" "next(x['id'] for x in d['data'] if x['projNo']=='$PROJ')")

# P2 加明细
req POST "/odn/constructions/$PID/items" "{\"facilityCode\":\"$FAC\"}"
[ "${HTTP_CODE:-}" = "200" ] && assert_eq "P2.code" "0" "$(jf "$BODY" "d['code']")" "加明细" || bad "P2" "HTTP=${HTTP_CODE:-} body=${BODY:-}"

# P3 开工 → 设施 IN_BUILD
req POST "/odn/constructions/$PID/start"
[ "${HTTP_CODE:-}" = "200" ] || bad "P3.http" "期望=200 实际=${HTTP_CODE:-}"
req GET "/odn/facilities/$FAC"
assert_eq "P3.lifecycle" "IN_BUILD" "$(jf "$BODY" "d['data']['lifecycleStatus']")" "设施随开工批量转施工中"

# P4 竣工 → ACCEPTED + 设施 IN_SERVICE
req POST "/odn/constructions/$PID/accept" "{\"note\":\"acc_ocn as-built 回填\"}"
[ "${HTTP_CODE:-}" = "200" ] || bad "P4.http" "期望=200 实际=${HTTP_CODE:-}"
req GET "/odn/facilities/$FAC"
assert_eq "P4.lifecycle" "IN_SERVICE" "$(jf "$BODY" "d['data']['lifecycleStatus']")" "竣工批量回填在网"

# P5 详情
req GET "/odn/constructions/$PID"
assert_eq "P5.status" "ACCEPTED" "$(jf "$BODY" "d['data']['project']['status']")" "单状态"
assert_eq "P5.items" "1" "$(jf "$BODY" "len(d['data']['items'])")" "明细数"
assert_eq "P5.note" "acc_ocn as-built 回填" "$(jf "$BODY" "d['data']['project']['asbuiltNote']")" "as-built 备注"

# P6 ACCEPTED 后加明细被拒
req POST "/odn/constructions/$PID/items" "{\"facilityCode\":\"$FAC\"}"
C=$(jf "$BODY" "d['code']")
if [ "$C" != "0" ]; then ok "P6" "终态加明细被拒 code=$C"; else bad "P6" "未被拒绝 body=$BODY"; fi

# P7 ACCEPTED 后再开工被拒
req POST "/odn/constructions/$PID/start"
C=$(jf "$BODY" "d['code']")
if [ "$C" != "0" ]; then ok "P7" "终态再开工被拒 code=$C"; else bad "P7" "未被拒绝 body=$BODY"; fi

# RES 收尾
RES=$(cleanup_data 2>/dev/null | tr '\n' ' ')
echo "RES: $RES"
assert_eq "RES.projects" "projects=0" "$(echo "$RES" | grep -o 'projects=[0-9]*' | head -1)" "施工单清理"
assert_eq "RES.facility" "facility=0" "$(echo "$RES" | grep -o 'facility=[0-9]*' | head -1)" "设施清理"

if [ "$FAIL_N" -eq 0 ]; then echo "[e2e-odn-construction RESULT: PASS]"; else echo "[e2e-odn-construction RESULT: FAIL ids:$FAILED_IDS]"; exit 1; fi