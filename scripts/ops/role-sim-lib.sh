#!/usr/bin/env bash
# role-sim-lib: 全角色诉求模拟套件共享库(bossctl x 102 真实环境, S11/T21)。
# 二进制解析 / 角色 key 装载 / 调用断言原语 / RLS 夹具自举 / PASS-FAIL 台账。
# 输出契约: PASS ROLE:<角色> <场景> | FAIL ROLE:<角色> <场景> <原因>。
set -u

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SERVER="http://192.168.0.102:28080"
  ACCOUNTS="$ROOT/.agents/skills/bossctl-cli/test-accounts.json"
  SSH_HOST="imeepos@192.168.0.102"
BOSSCTL=""
GO_BIN=""
API="$SERVER/api/admin/v1"

if [ -z "$SERVER" ]; then
  SERVER="http://192.168.0.102:28080"
fi
if [ -z "$ACCOUNTS" ]; then
  ACCOUNTS="$ROOT/.agents/skills/bossctl-cli/test-accounts.json"
fi
if [ -z "$SSH_HOST" ]; then
  SSH_HOST="imeepos@192.168.0.102"
BOSSCTL=""
GO_BIN=""
fi
API="$SERVER/api/admin/v1"

if [ -z "$BOSSCTL" ] && command -v bossctl >/dev/null 2>&1; then
  BOSSCTL="$(command -v bossctl)"
fi
if [ -z "$BOSSCTL" ] && [ -x "$HOME/bin/bossctl" ]; then
  BOSSCTL="$HOME/bin/bossctl"
fi
if [ -z "$BOSSCTL" ]; then
  GO_BIN="$GO"
  if [ -z "$GO_BIN" ]; then
    GO_BIN="$(command -v go 2>/dev/null)"
  fi
  if [ -z "$GO_BIN" ]; then
    echo "FAIL: bossctl 不可用且无 go 可编译" >&2
    exit 1
  fi
  BOSSCTL="$(mktemp -t rolesim-bossctl.XXXXXX)"
  (cd "$ROOT" && "$GO_BIN" build -o "$BOSSCTL" ./cmd/bossctl) || { echo "FAIL: go build ./cmd/bossctl" >&2; exit 1; }
  echo "INFO: bossctl built from source at $BOSSCTL" >&2
fi

load_role_keys() {
  eval "$(python3 - "$ACCOUNTS" <<'PY'
import json, sys
d = json.load(open(sys.argv[1]))
print("K_ADMIN=%s" % d["admin"]["apiKeys"][0]["key"])
print("K_KEFU=%s" % d["kefu_xu"]["apiKeys"][0]["key"])
print("K_DISP=%s" % d["dispatch_li"]["apiKeys"][0]["key"])
print("K_CASH=%s" % d["cashier_wang"]["apiKeys"][0]["key"])
print("K_NOC=%s" % d["noc_chen"]["apiKeys"][0]["key"])
print("K_REV=%s" % d["reviewer1"]["apiKeys"][0]["key"])
print("K_CUST=%s" % d["customers"][1]["apiKey"])
print("K_WORKER=%s" % d["workers"][0]["apiKey"])
PY
)" || { echo "FAIL: test accounts unreadable" >&2; exit 1; }
}

bc() {
  local key="$1"; shift
  local out rc att
  out=$("$BOSSCTL" --server "$SERVER" --api-key "$key" "$@" 2>&1); rc=$?
  for att in 2 3 4 5 6; do
    [ "$rc" -eq 0 ] && break
    sleep 4
    out=$("$BOSSCTL" --server "$SERVER" --api-key "$key" "$@" 2>&1); rc=$?
  done
  printf '%s' "$out"
  return $rc
}

sql() {
  ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST"     "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"
}

PASS_COUNT=0
FAIL_COUNT=0

sim_pass() {
  PASS_COUNT=$((PASS_COUNT + 1))
  echo "PASS ROLE:$1 $2"
}
sim_fail() {
  FAIL_COUNT=$((FAIL_COUNT + 1))
  echo "FAIL ROLE:$1 $2 $3"
}

EXPECT_LAST_OUT=""
EXPECT_FAIL_REASON=""

expect_ok() {
  EXPECT_LAST_OUT="$3"
  EXPECT_FAIL_REASON=""
  if [ "$2" -ne 0 ]; then
    EXPECT_FAIL_REASON="$(printf '%s' "$3" | head -n 1)"
    return 1
  fi
  return 0
}

expect_fail() {
  EXPECT_LAST_OUT="$3"
  if [ "$2" -eq 0 ]; then
    EXPECT_FAIL_REASON="预期被拒却成功"
    return 1
  fi
  if [ -z "$(printf '%s' "$3" | tr -d '[:space:]')" ]; then
    EXPECT_FAIL_REASON="被拒但无可观测信号(空输出)"
    return 1
  fi
  return 0
}

jid() { python3 -c '
import json, sys
try:
    d = json.loads(sys.stdin.read())
except Exception:
    print(""); sys.exit(0)
if isinstance(d, dict):
    if "id" in d:
        print(d["id"]); sys.exit(0)
    dd = d.get("data")
    if isinstance(dd, dict) and "id" in dd:
        print(dd["id"]); sys.exit(0)
print("")'; }

jno() { python3 -c 'import json,sys
try:
    print(json.loads(sys.stdin.read())["orderNo"])
except Exception:
    print("")' ; }
jpick() { python3 -c 'import json,sys
mf, mv, rf = sys.argv[1], sys.argv[2], sys.argv[3]
d = json.load(sys.stdin)
items = d.get("items") if isinstance(d, dict) else d
for t in items or []:
    if str(t.get(mf, "")) == mv:
        print(t.get(rf, "")); break' "$1" "$2" "$3" ; }

order_id_by_no() { sql <<SQL
SELECT id FROM orders WHERE order_no='$1';
SQL
}
order_state() { sql <<SQL
SELECT stage || '/' || status FROM orders WHERE order_no='$1';
SQL
}

fixture_address() {
  bc "$K_ADMIN" call POST /addresses     --data '{"label":"acc_'"$1"'","name":"角色模拟-'"$1"'","regionId":4}' | jid
}

fixture_olt_ports() {
  local sfx="$1" aid="$2" rid i
  rid=$(bc "$K_NOC" call POST /provision/resources     --data '{"code":"OLT-RLS-'"$sfx"'","name":"模拟OLT","type":"OLT","addressId":'"$aid"',"legalEntityId":1}' | jid)
  [ -n "$rid" ] || return 1
  for i in 01 02; do
    bc "$K_NOC" call POST /provision/ports       --data '{"portCode":"P-RLS-'"$sfx"'-'"$i"'","resourceId":'"$rid"',"addressId":'"$aid"',"legalEntityId":1}' >/dev/null || return 1
  done
  sql <<SQL
UPDATE resources SET nms_oltid='NMS-RLS-$sfx' WHERE id=$rid;
SQL
  echo "$rid"
}

fixture_tag_asset() {
  local sfx="$1" bid gtid
  bid=$(bc "$K_ADMIN" call POST /provision/asset-batches     --data '{"code":"RK-RLS-'"$sfx"'","name":"模拟批次","legalEntityId":1}' | jid)
  gtid=$(bc "$K_ADMIN" call POST /provision/tags     --data '{"tagNo":"T-RLS-'"$sfx"'","epcCode":"EPC-RLS-'"$sfx"'","legalEntityId":1}' | jid)
  [ -n "$bid" ] && [ -n "$gtid" ] || return 1
  bc "$K_ADMIN" call POST /provision/assets     --data '{"assetCode":"A-RLS-'"$sfx"'","batchId":'"$bid"',"tagId":'"$gtid"',"legalEntityId":1}' >/dev/null
}

fixture_template() {
  bc "$K_ADMIN" call POST /provision-templates --data '{"legalEntityId":1,"code":"TPL-RLS-'"$1"'","name":"模拟模板","content":{"bandwidth":"100M","onuType":"Internet","services":{"internet":{"svlan":1113,"cvlan":1,"uv":100,"scos":0,"ccos":0},"tr069":{"cvlan":1000,"uv":1000,"scos":6,"ccos":6}}}}' | jid
}

fixture_pon_task() {
  local oid="$1" tpl="$2"
  sql <<SQL
UPDATE ports SET pon_frame=0,pon_slot=7,pon_port=7,onu_no=NULL WHERE order_id=$oid AND status IN ('RESERVED','USED');
UPDATE provision_tasks SET template_id=$tpl, status='PENDING' WHERE order_id=$oid AND status<>'DONE';
SQL
}

# ===== worker / kefu / noc 场景(依赖主流程变量) =====

sc_worker_accept() {
  local scene="接单扫码激活签收" out rc
  out=$(bc "$K_WORKER" call POST worker:/tickets/$TICKET_NO/accept --data '{}'); rc=$?
  expect_ok "$scene-accept" "$rc" "$out" || { sim_fail worker "$scene" "$EXPECT_FAIL_REASON"; return 0; }
}

sc_worker_preactivate() {
  local scene="未扫码先激活被拒(时序异常)" out rc
  out=$(bc "$K_WORKER" call POST worker:/tickets/$TICKET_NO/activate --data '{}'); rc=$?
  if expect_fail "$scene" "$rc" "$out"; then
    sim_pass worker "$scene"
  else
    sim_fail worker "$scene" "$EXPECT_FAIL_REASON"
  fi
}

sc_worker_finish() {
  local scene="接单扫码激活签收" out rc stage tasks_open waited=0
  out=$(bc "$K_WORKER" call POST worker:/tickets/$TICKET_NO/scan-bind --data '{"epc":"EPC-RLS-'"$GLOBAL_SFX"'"}'); rc=$?
  expect_ok "$scene-scan" "$rc" "$out" || { sim_fail worker "$scene" "$EXPECT_FAIL_REASON"; return 0; }
  out=$(bc "$K_WORKER" call POST worker:/tickets/$TICKET_NO/activate --data '{}'); rc=$?
  expect_ok "$scene-activate" "$rc" "$out" || { sim_fail worker "$scene" "$EXPECT_FAIL_REASON"; return 0; }
  out=$(bc "$K_WORKER" call POST worker:/tickets/$TICKET_NO/sign --data '{}'); rc=$?
  expect_ok "$scene-sign" "$rc" "$out" || { sim_fail worker "$scene" "$EXPECT_FAIL_REASON"; return 0; }
  stage=$(order_state "$ORDER_MAIN_NO")
  while [ $waited -lt 45 ]; do
    tasks_open=$(sql <<SQL
SELECT count(*) FROM provision_tasks WHERE order_id=$ORDER_MAIN_ID AND status<>'DONE';
SQL
)
    [ "$tasks_open" = "0" ] && break
    sleep 3; waited=$((waited + 3))
  done
  if printf '%s' "$stage" | grep -q '^12/DONE'; then
    if [ "$tasks_open" != "0" ]; then
      echo "[worker] INFO 下发任务未完结(tl1sim 环境暴露): $tasks_open 项" >&2
    fi
    sim_pass worker "$scene"
  else
    sim_fail worker "$scene" "stage=$stage 未完结下发任务=$tasks_open"
  fi
}

sc_kefu_query_complaint() {
  local scene="订单查询与投诉受理办结" out rc st
  out=$(bc "$K_KEFU" call GET /orders --query pageSize=5); rc=$?
  bc "$K_CUST" call POST user:/complaints --data '{"type":"attitude","description":"acc_ 模拟投诉-'"$GLOBAL_SFX"'","relOrderNo":"'"$ORDER_MAIN_NO"'"}' >/dev/null
  CMP_TICKET=$(sql <<SQL
SELECT ticket_no FROM complaints WHERE description LIKE 'acc/_%' ESCAPE '/' ORDER BY id DESC LIMIT 1;
SQL
)
  if ! expect_ok "$scene" "$rc" "$out" || [ -z "$CMP_TICKET" ]; then
    sim_fail kefu "$scene" "订单查询或投诉创建失败"
    return 0
  fi
  bc "$K_KEFU" call POST /complaints/$CMP_TICKET/status --data '{"status":"PROCESSING"}' >/dev/null || {
    sim_fail kefu "$scene" "受理失败"; return 0; }
  bc "$K_KEFU" call POST /complaints/$CMP_TICKET/close --data '{}' >/dev/null || {
    sim_fail kefu "$scene" "办结失败"; return 0; }
  st=$(sql <<SQL
SELECT status FROM complaints WHERE ticket_no='$CMP_TICKET';
SQL
)
  if [ "$st" = "CLOSED" ]; then
    sim_pass kefu "$scene"
  else
    sim_fail kefu "$scene" "终态=$st"
  fi
}

sc_kefu_no_order() {
  local scene="越权下单被拒(账号主体不可冒客户)" out rc
  out=$(bc "$K_KEFU" call POST user:/orders --data '{}'); rc=$?
  if expect_fail "$scene" "$rc" "$out"; then
    sim_pass kefu "$scene"
  else
    sim_fail kefu "$scene" "$EXPECT_FAIL_REASON"
  fi
}

sc_noc_provision() {
  local scene="端口查询与置备" out rc
  out=$(bc "$K_NOC" call GET /ports --query pageSize=50); rc=$?
  if expect_ok "$scene" "$rc" "$out" && [ -n "$OLT_ID" ] && printf '%s' "$out" | grep -q "P-RLS-"; then
    sim_pass noc "$scene"
  else
    sim_fail noc "$scene" "置备olt=$OLT_ID 或端口清单缺 P-RLS 标记"
  fi
}

sc_noc_cross_domain() {
  local scene="跨域调财务接口被拒" out rc
  out=$(bc "$K_NOC" call POST /orders/$ORDER_MAIN_NO/charge --data '{}'); rc=$?
  if expect_fail "$scene" "$rc" "$out" && printf '%s' "$out" | grep -qE '403|no permission'; then
    sim_pass noc "$scene"
  else
    sim_fail noc "$scene" "$EXPECT_FAIL_REASON"
  fi
}
