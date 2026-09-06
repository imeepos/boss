#!/usr/bin/env bash
# verify-oss-pon-path-e2e.sh -- PON 端到端链路反查端到端实测(P5-W2,102 真实部署,禁 mock)。
# 断言:
#   P0 造数: acc_ppath 前缀族(幂等重跑):OLT→分光器→端口1(pon 0/1/1,USED+订单占用);
#           SPL2 无上级→端口2(未配 PON);OLT 直挂端口3(两跳终止组)。
#   P1 完整链路: GET /ports/acc_ppath-P1/path → code=0;hops 顺序 PORT→SPLITTER→PON_PORT→OLT;
#               编码/状态/占用(orderNo)逐项断言;complete=true。
#   P2 断链场景: 分光器无上级+端口未配 PON → code=0(HTTP 200 非报错);missing 集合=
#               (PON_PORT,PON_PORT_UNASSIGNED)+(OLT,SPLITTER_NO_PARENT);complete=false;
#               全链路编码不含 acc_ppath-OLT(禁猜链补链)。
#   P3 直挂 OLT: 端口3 → 两跳 PORT→OLT,complete=true(无分光器/PON 跳)。
#   P4 ID 入口: GET /ports/{id}/path(资源标识)与 P1 同构。
#   P5 端口不存在: code=40400(错误而非补链)。
#   RES 收尾清理造数(acc_ppath 前缀)后残留断言为零(脚本可重复执行)。
# 信号: 每条断言独立输出 "PASS: [编号] ..." / "FAIL: [编号] ...";收尾
#   "E2E-OSS-PON-PATH RESULT: ..." 可 grep。
# 用法: scripts/e2e/verify-oss-pon-path-e2e.sh [BASE_URL]
# 环境: BASE_URL ADMIN_API_KEY SSH_HOST SKIP_CLEANUP=1
# 依赖: curl python3 ssh(102 免密);鉴权 X-API-Key(test-accounts.json admin key)。
# 注意: 依赖 P5-W2 链路反查 API(GET /ports/:portId/path)已随部署生效(Lead 统一收口后验收)。
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

sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }
sqlval() { sql < <(echo "$1") | tr -d '[:space:]'; }

FAIL_REASON=""
# req METHOD PATH [JSON] -> HTTP_CODE/BODY(业务失败也保留响应供断言)。
req() {
  local method="$1" path="$2" body="${3:-}" out
  out=$(curl -sS -m 30 -w $'\n%{http_code}' -X "$method" "$API$path" -H "X-API-Key: $KEY" \
    -H "Content-Type: application/json" ${body:+-d "$body"}) || { FAIL_REASON="curl $path"; return 1; }
  HTTP_CODE=$(printf '%s\n' "$out" | tail -n1 | tr -d '[:space:]')
  BODY=$(printf '%s\n' "$out" | sed '$d')
}

jf() { python3 -c "import json,sys;d=json.loads(sys.argv[1]);print(eval(sys.argv[2]))" "$1" "$2"; }

PASS_N=0; FAIL_N=0; FAILED_IDS=""
ok() { PASS_N=$((PASS_N+1)); echo "PASS: [$1] $2"; }
bad() {
  FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"
  echo "FAIL: [$1] $2" >&2
  echo "[e2e-oss-pon-path] ASSERTION FAILED id=$1 detail=$2" >&2
}
assert_eq() { if [ "$2" = "$3" ]; then ok "$1" "$2 ($4)"; else bad "$1" "期望=$2 实际=$3 上下文=$4"; fi; }

cleanup_data() {
  echo "清尾: 造数自清理(acc_ppath 前缀,幂等)" >&2
  sql <<'SQL'
DELETE FROM ports WHERE port_code LIKE 'acc_ppath-%';
DELETE FROM orders WHERE order_no LIKE 'acc_ppath-%';
DELETE FROM resources WHERE code LIKE 'acc_ppath-%';
SELECT 'ports=' || count(*) FROM ports WHERE port_code LIKE 'acc_ppath-%'
UNION ALL SELECT 'resources=' || count(*) FROM resources WHERE code LIKE 'acc_ppath-%'
UNION ALL SELECT 'orders=' || count(*) FROM orders WHERE order_no LIKE 'acc_ppath-%';
SQL
}

seed_data() {
  cleanup_data >/dev/null || { FAIL_REASON="pre-clean"; return 1; }
  local ent addr olt spl spl2 ord p1 p2 p3
  ent=$(sqlval "SELECT min(id) FROM legal_entities;")
  addr=$(sqlval "SELECT min(id) FROM addresses;")
  if [ -z "$ent" ] || [ -z "$addr" ]; then FAIL_REASON="no legal_entities/addresses row"; return 1; fi
  olt=$(sqlval "INSERT INTO resources(legal_entity_id, code, name, type, parent_id, address_id, status) VALUES ($ent, 'acc_ppath-OLT', '链路e2e-OLT', 'OLT', NULL, $addr, 'ONLINE') RETURNING id;")
  spl=$(sqlval "INSERT INTO resources(legal_entity_id, code, name, type, parent_id, address_id, status) VALUES ($ent, 'acc_ppath-SPL', '链路e2e-分光器', 'SPLITTER', $olt, $addr, 'ONLINE') RETURNING id;")
  spl2=$(sqlval "INSERT INTO resources(legal_entity_id, code, name, type, parent_id, address_id, status) VALUES ($ent, 'acc_ppath-SPL2', '链路e2e-断链分光器', 'SPLITTER', NULL, $addr, 'ONLINE') RETURNING id;")
  ord=$(sqlval "INSERT INTO orders(order_no, customer_id, offer_id, address_id, stage, status, channel_id, legal_entity_id) VALUES ('acc_ppath-ORD', 0, 0, $addr, 1, 'PENDING', 0, $ent) RETURNING id;")
  p1=$(sqlval "INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, order_id, status, pon_frame, pon_slot, pon_port) VALUES ('acc_ppath-P1', 'acc_ppath-Q1', $spl, $ent, '链路e2e法人', $addr, 0, 'e2e', $ord, 'USED', 0, 1, 1) RETURNING id;")
  p2=$(sqlval "INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, order_id, status) VALUES ('acc_ppath-P2', 'acc_ppath-Q2', $spl2, $ent, '链路e2e法人', $addr, 0, 'e2e', NULL, 'IDLE') RETURNING id;")
  p3=$(sqlval "INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, order_id, status) VALUES ('acc_ppath-P3', 'acc_ppath-Q3', $olt, $ent, '链路e2e法人', $addr, 0, 'e2e', NULL, 'IDLE') RETURNING id;")
  if [ -z "$olt" ] || [ -z "$spl" ] || [ -z "$spl2" ] || [ -z "$ord" ] || [ -z "$p1" ] || [ -z "$p2" ] || [ -z "$p3" ]; then
    FAIL_REASON="seed insert"; return 1
  fi
  echo "$olt $spl $spl2 $ord $p1 $p2 $p3"
}

echo "PON 端到端链路反查端到端实测 @ $BASE_URL"
SEED=$(seed_data) || { bad "P0" "造数失败: $FAIL_REASON"; cleanup_data >/dev/null 2>&1; exit 1; }
read -r OLT_ID SPL_ID SPL2_ID ORD_ID P1_ID P2_ID P3_ID <<< "$SEED"
ok "P0" "造数基线 acc_ppath olt=$OLT_ID spl=$SPL_ID spl2=$SPL2_ID ord=$ORD_ID p1=$P1_ID p2=$P2_ID p3=$P3_ID"

# P1 完整链路(端口编码入口)
if ! req GET "/ports/acc_ppath-P1/path"; then bad "P1" "请求失败: $FAIL_REASON"; fi
if [ "${HTTP_CODE:-}" = "200" ]; then ok "P1.http" "HTTP $HTTP_CODE"; else bad "P1.http" "期望=200 实际=${HTTP_CODE:-} body=${BODY:-}"; fi
if [ "${HTTP_CODE:-}" = "200" ]; then
  assert_eq "P1.code" "0" "$(jf "$BODY" "d['code']")" "业务码"
  assert_eq "P1.kinds" "['PORT', 'SPLITTER', 'PON_PORT', 'OLT']" "$(jf "$BODY" "[h['kind'] for h in d['data']['hops']]")" "逐跳类型顺序"
  assert_eq "P1.codes" "['acc_ppath-P1', 'acc_ppath-SPL', 'NA-0-1-1', 'acc_ppath-OLT']" "$(jf "$BODY" "[h['code'] for h in d['data']['hops']]")" "逐跳编码"
  assert_eq "P1.status" "['USED', 'ONLINE', '', 'ONLINE']" "$(jf "$BODY" "[h['status'] for h in d['data']['hops']]")" "逐跳状态(PON 无独立状态)"
  assert_eq "P1.occupy" "acc_ppath-ORD" "$(jf "$BODY" "d['data']['hops'][0]['occupiedBy']['orderNo']")" "端口占用订单引用"
  assert_eq "P1.complete" "True" "$(jf "$BODY" "d['data']['complete']")" "链路完整"
fi

# P2 断链场景(分光器无上级+端口未配 PON)
if ! req GET "/ports/acc_ppath-P2/path"; then bad "P2" "请求失败: $FAIL_REASON"; fi
if [ "${HTTP_CODE:-}" = "200" ]; then ok "P2.http" "HTTP $HTTP_CODE(断链返回视图不报错)"; else bad "P2.http" "期望=200 实际=${HTTP_CODE:-} body=${BODY:-}"; fi
if [ "${HTTP_CODE:-}" = "200" ]; then
  assert_eq "P2.code" "0" "$(jf "$BODY" "d['code']")" "业务码"
  assert_eq "P2.complete" "False" "$(jf "$BODY" "d['data']['complete']")" "断链标记"
  assert_eq "P2.missing" "[('PON_PORT', 'PON_PORT_UNASSIGNED'), ('OLT', 'SPLITTER_NO_PARENT')]" "$(jf "$BODY" "[(h['kind'], h['reason']) for h in d['data']['hops'] if h['missing']]")" "断点原因码"
  P2_CODES=$(jf "$BODY" "[h['code'] for h in d['data']['hops'] if h['code']]")
  case "$P2_CODES" in
    *acc_ppath-OLT*) bad "P2.nofill" "断链链路出现补链编码: $P2_CODES" ;;
    *) ok "P2.nofill" "无补链编码 ($P2_CODES)" ;;
  esac
fi

# P3 端口直挂 OLT(两跳终止)
if ! req GET "/ports/acc_ppath-P3/path"; then bad "P3" "请求失败: $FAIL_REASON"; fi
if [ "${HTTP_CODE:-}" = "200" ]; then
  assert_eq "P3.kinds" "['PORT', 'OLT']" "$(jf "$BODY" "[h['kind'] for h in d['data']['hops']]")" "直挂两跳"
  assert_eq "P3.complete" "True" "$(jf "$BODY" "d['data']['complete']")" "直挂完整"
else bad "P3" "HTTP=${HTTP_CODE:-} body=${BODY:-}"; fi

# P4 端口 ID 入口(资源标识)
if ! req GET "/ports/$P1_ID/path"; then bad "P4" "请求失败: $FAIL_REASON"; fi
if [ "${HTTP_CODE:-}" = "200" ]; then
  assert_eq "P4.portCode" "acc_ppath-P1" "$(jf "$BODY" "d['data']['portCode']")" "ID 反查同端口"
  assert_eq "P4.complete" "True" "$(jf "$BODY" "d['data']['complete']")" "ID 反查完整"
else bad "P4" "HTTP=${HTTP_CODE:-} body=${BODY:-}"; fi

# P5 端口不存在
if ! req GET "/ports/acc_ppath-NONE/path"; then bad "P5" "请求失败: $FAIL_REASON"; fi
assert_eq "P5.code" "40400" "$(jf "$BODY" "d['code']")" "不存在返回 40400(非断链视图)"

# RES 收尾清理 + 残留断言
if [ -z "${SKIP_CLEANUP:-}" ]; then
  RES_OUT=$(cleanup_data) || bad "RES" "清理执行失败"
  RES_COUNTS=$(printf '%s' "$RES_OUT" | tr -d '[:space:]')
  assert_eq "RES.residual" "ports=0resources=0orders=0" "$RES_COUNTS" "造数残留为零"
else
  echo "SKIP_CLEANUP=1 跳过清理(调试用,记得手动清)" >&2
fi

echo "E2E-OSS-PON-PATH RESULT: PASS=$PASS_N FAIL=$FAIL_N$FAILED_IDS"
[ "$FAIL_N" -eq 0 ]
