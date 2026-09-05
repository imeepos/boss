#!/usr/bin/env bash
# role-sim-lib: 全角色诉求模拟套件共享库(bossctl x 102 真实环境, 2026-09-05 S11/T21)。
# 职责: 二进制解析 / 角色 key 装载 / 调用与断言原语 / TL1 夹具自举 / PASS-FAIL 台账。
# 契约: 场景输出固定为
#   PASS ROLE:<role> <scene>  |  FAIL ROLE:<role> <scene> <reason>
# 八种角色标记: customer/worker/kefu/dispatch/cashier/noc/reviewer/admin。
set -u

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SERVER="${BOSS_SERVER:-http://192.168.0.102:28080}"
ACCOUNTS="${BOSSCTL_TEST_ACCOUNTS:-$ROOT/.agents/skills/bossctl-cli/test-accounts.json}"
SSH_HOST="${SSH_HOST:-imeepos@192.168.0.102}"
API="$SERVER/api/admin/v1"

# 二进制解析优先级同 bossctl-acceptance.sh: env > PATH > ~/bin > 源码临时编译。
# 禁止回落到技能 assets 里过期的预编译产物(落后 102 路由会误判)。
if [ -z "${BOSSCTL:-}" ] && command -v bossctl >/dev/null 2>&1; then
  BOSSCTL="$(command -v bossctl)"
fi
if [ -z "${BOSSCTL:-}" ] && [ -x "$HOME/bin/bossctl" ]; then
  BOSSCTL="$HOME/bin/bossctl"
fi
if [ -z "${BOSSCTL:-}" ]; then
  GO_BIN="${GO:-$(command -v go 2>/dev/null || true)}"
  [ -n "$GO_BIN" ] || { echo "FAIL: bossctl 不可用且无 go 可编译" >&2; exit 1; }
  BOSSCTL="$(mktemp -t rolesim-bossctl.XXXXXX)"
  (cd "$ROOT" && "$GO_BIN" build -o "$BOSSCTL" ./cmd/bossctl) || { echo "FAIL: go build ./cmd/bossctl" >&2; exit 1; }
  echo "INFO: bossctl built from source at $BOSSCTL" >&2
fi

# 角色 key 一律取自测试账号唯一事实源; 漂移以 102 线上 /auth/me 实测为准。
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

# bc <key> <bossctl args...>: 带身份调用, stdout+stderr 合流, 退出码透传。
bc() {
  local key="$1"; shift
  "$BOSSCTL" --server "$SERVER" --api-key "$key" "$@" 2>&1
}

# sql: 102 夹具 SQL(姿势同 mainchain-acceptance.sh, 仅限无管理面字段)。
sql() {
  ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST"     "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"
}

PASS_COUNT=0
FAIL_COUNT=0
FAIL_SUMMARY=""

# sim_pass/sim_fail: 场景唯一出口, 保证输出契约可 grep。
sim_pass() {
  PASS_COUNT=$((PASS_COUNT + 1))
  echo "PASS ROLE:$1 $2"
}
sim_fail() {
  FAIL_COUNT=$((FAIL_COUNT + 1))
  FAIL_SUMMARY="$FAIL_SUMMARY
FAIL ROLE:$1 $2 $3"
  echo "FAIL ROLE:$1 $2 $3"
}

EXPECT_LAST_OUT=""
EXPECT_FAIL_REASON=""

# expect_ok <label> <rc> <out>: 调用必须成功。
expect_ok() {
  EXPECT_LAST_OUT="$3"
  if [ "$2" -ne 0 ]; then
    EXPECT_FAIL_REASON="$1: $(printf '%s' "$3" | head -n 1)"
    return 1
  fi
  return 0
}

# expect_fail <label> <rc> <out>: 调用必须被拒且留可观测信号(错误文本非空)。
expect_fail() {
  EXPECT_LAST_OUT="$3"
  if [ "$2" -eq 0 ]; then
    EXPECT_FAIL_REASON="$1: 预期被拒却成功"
    return 1
  fi
  if [ -z "$(printf '%s' "$3" | tr -d '[:space:]')" ]; then
    EXPECT_FAIL_REASON="$1: 被拒但无可观测信号(空输出)"
    return 1
  fi
  return 0
}

# jid/jno: bossctl 已解包 data, 这里做轻量取值。
jid() { python3 -c "import json,sys
try: print(json.load(sys.stdin)['id'])
except Exception: print('')"; }
jno() { python3 -c "import json,sys
try: print(json.load(sys.stdin)['orderNo'])
except Exception: print('')"; }
jpick() { python3 -c "
import json, sys
d = json.load(sys.stdin)
items = d.get('items') if isinstance(d, dict) else d
for t in items or []:
    if str(t.get(sys.argv[1], '')) == sys.argv[2]: print(t.get(sys.argv[3], '')); break" "$1" "$2" "$3"; }

# order_id_by_no / order_state: 订单主键与状态机只读断言(先查库再复核)。
order_id_by_no() { sql <<SQL
SELECT id FROM orders WHERE order_no='$1';
SQL
}
order_state() { sql <<SQL
SELECT stage || '/' || status FROM orders WHERE order_no='$1';
SQL
}

# fixture_address <suffix>: 建验收地址(regionId=4 与师傅 6 同区, 避免跨区拒派)。
fixture_address() {
  bc "$K_ADMIN" call POST /addresses     --data '{"label":"acc_'"$1"'","name":"验收地址-'"$1"'","regionId":4}' | jid
}

# fixture_olt_ports <suffix> <addressId>: noc 置备 OLT+2端口并回填 nms_oltid(TL1 就绪)。
fixture_olt_ports() {
  local sfx="$1" aid="$2" rid i
  rid=$(bc "$K_NOC" call POST /provision/resources     --data '{"code":"OLT-ACC-'"$sfx"'","name":"模拟OLT","type":"OLT","addressId":'"$aid"',"legalEntityId":1}' | jid)
  [ -n "$rid" ] || return 1
  for i in 01 02; do
    bc "$K_NOC" call POST /provision/ports       --data '{"portCode":"P-ACC-'"$sfx"'-'"$i"'","resourceId":'"$rid"',"addressId":'"$aid"',"legalEntityId":1}' >/dev/null || return 1
  done
  sql <<SQL
UPDATE resources SET nms_oltid='NMS-ACC-$sfx' WHERE id=$rid;
SQL
  echo "$rid"
}

# fixture_tag_asset <suffix>: 批次+标签+资产(环节5 预绑定与环节9 扫码比对源)。
fixture_tag_asset() {
  local sfx="$1" bid gtid
  bid=$(bc "$K_ADMIN" call POST /provision/asset-batches     --data '{"code":"RK-ACC-'"$sfx"'","name":"模拟批次","legalEntityId":1}' | jid)
  gtid=$(bc "$K_ADMIN" call POST /provision/tags     --data '{"tagNo":"T-ACC-'"$sfx"'","epcCode":"EPC-ACC-'"$sfx"'","legalEntityId":1}' | jid)
  [ -n "$bid" ] && [ -n "$gtid" ] || return 1
  bc "$K_ADMIN" call POST /provision/assets     --data '{"assetCode":"A-ACC-'"$sfx"'","batchId":'"$bid"',"tagId":'"$gtid"',"legalEntityId":1}' >/dev/null
}

# fixture_template <suffix>: TL1 内容模板(环节7 preConfigOLT 必达)。
fixture_template() {
  bc "$K_ADMIN" call POST /provision-templates --data '{"legalEntityId":1,"code":"TPL-ACC-'"$1"'","name":"模拟模板","content":{"bandwidth":"100M","onuType":"Internet","services":{"internet":{"svlan":1113,"cvlan":1,"uv":100,"scos":0,"ccos":0},"tr069":{"cvlan":1000,"uv":1000,"scos":6,"ccos":6}}}}' | jid
}

# fixture_pon_task <orderId> <templateId>: 预占端口回填 PON 三维 + 预建环节7 PENDING 任务
# (task_no 同号幂等, 姿势同 mainchain-acceptance.sh, 保证下发任务终态 DONE)。
fixture_pon_task() {
  local oid="$1" tpl="$2" loid
  loid=$(sql <<SQL
SELECT id FROM lo_accounts WHERE customer_id=214 ORDER BY id LIMIT 1;
SQL
)
  [ -n "$loid" ] || return 1
  sql <<SQL
UPDATE ports SET pon_frame=0,pon_slot=7,pon_port=$RANDOM,onu_no=NULL WHERE order_id=$oid AND status='RESERVED';
INSERT INTO provision_tasks(task_no,order_id,stage_event,lo_account_id,template_id,status) VALUES('PRV-O$oid',$oid,'preConfigOLT',$loid,$tpl,'PENDING');
SQL
}

# ===== 场景后段: worker/kefu/noc(依赖主流程变量) =====

WORKER_EARLY=0
WORKER_PRE_OUT=""
# sc_worker_preactivate: 未扫码先激活(时序异常)必须被拒。
sc_worker_preactivate() {
	local scene="未扫码先激活被拒(时序异常)" out rc
	out=$(bc "$K_WORKER" call POST worker:/tickets/$TICKET_NO/activate --data '{}'); rc=$?
	if expect_fail "$scene" "$rc" "$out"; then
		WORKER_EARLY=1
		sim_pass worker "未扫码先激活被拒(时序异常)"
	else
		WORKER_PRE_OUT=$EXPECT_FAIL_REASON
		sim_fail worker "未扫码先激活被拒(时序异常)" "$WORKER_PRE_OUT"
	fi
}
# sc_worker_chain: accept -> scan-bind -> activate -> sign -> 下发任务终态轮询。
sc_worker_accept() {
	local scene="接单扫码激活签收" out rc
	out=$(bc "$K_WORKER" call POST worker:/tickets/$TICKET_NO/accept --data '{}'); rc=$?
	expect_ok "$scene-accept" "$rc" "$out" || { sim_fail worker "$scene" "$EXPECT_FAIL_REASON"; return 0; }
}
sc_worker_finish() {
	local scene="接单扫码激活签收" out rc stage tasks_open waited=0
	SCAN_EPC=$(sql <<SQL
SELECT epc_code FROM tags WHERE bound_asset_id IS NOT NULL AND epc_code='EPC-ACC-$GLOBAL_SFX' LIMIT 1;
SQL
	)
	if [ -z "$SCAN_EPC" ]; then
		SCAN_EPC=$(sql <<SQL
SELECT epc_code FROM tags WHERE bound_asset_id IS NOT NULL ORDER BY id DESC LIMIT 1;
SQL
	)
	fi
	echo "[debug] scan ticket=$TICKET_NO epc=$SCAN_EPC" >&2
	out=$(bc "$K_WORKER" call POST worker:/tickets/$TICKET_NO/scan-bind --data '{"epc":"'"$SCAN_EPC"'"}'); rc=$?
	expect_ok "$scene-scan" "$rc" "$out" || { sim_fail worker "$scene" "$EXPECT_FAIL_REASON"; return 0; }
	out=$(bc "$K_WORKER" call POST worker:/tickets/$TICKET_NO/activate --data '{}'); rc=$?
	expect_ok "$scene-activate" "$rc" "$out" || { sim_fail worker "$scene" "$EXPECT_FAIL_REASON"; return 0; }
	out=$(bc "$K_WORKER" call POST worker:/tickets/$TICKET_NO/sign --data '{}'); rc=$?
	expect_ok "$scene-sign" "$rc" "$out" || { sim_fail worker "$scene" "$EXPECT_FAIL_REASON"; return 0; }
	stage=$(order_state "$ORDER_MAIN_NO")
	while [ $waited -lt 60 ]; do
		tasks_open=$(sql <<SQL
	SELECT count(*) FROM provision_tasks WHERE order_id=$ORDER_MAIN_ID AND status<>'DONE';
SQL
	)
		[ "$tasks_open" = "0" ] && break
		sleep 3; waited=$((waited + 3))
	done
	if printf '%s' "$stage" | grep -q '^12/DONE' && [ "$tasks_open" = "0" ]; then
		sim_pass worker "$scene"
	else
		sim_fail worker "$scene" "stage=$stage 未完结下发任务=$tasks_open"
	fi

}
sc_kefu_query_complaint() {
  local scene="订单查询与投诉受理办结" out rc cno rc2 ctk st
  out=$(bc "$K_KEFU" call GET /orders --query pageSize=5); rc=$?
  cno=$(bc "$K_CUST" call POST user:/complaints --data '{"type":"attitude","description":"acc_ 模拟投诉-'"$GLOBAL_SFX"'","relOrderNo":"'"$ORDER_MAIN_NO"'"}'); rc2=$?
  CMP_TICKET=$(sql <<SQL
SELECT ticket_no FROM complaints WHERE description LIKE 'acc/_%' ESCAPE '/' ORDER BY id DESC LIMIT 1;
SQL
)
  if ! expect_ok "$scene" "$rc" "$out" || [ -z "$CMP_TICKET" ]; then
    sim_fail kefu "$scene" "查询或投诉创建失败"
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
  out=$(bc "$K_KEFU" call POST user:/orders --data '{"productId":"101","addressId":"'"$ADDR_ID"'","channelId":"102"}'); rc=$?
  if expect_fail "$scene" "$rc" "$out"; then
    sim_pass kefu "$scene"
  else
    sim_fail kefu "$scene" "$EXPECT_FAIL_REASON"
  fi
}

sc_noc_provision() {
  local scene="端口查询与置备" out rc
  out=$(bc "$K_NOC" call GET /ports --query pageSize=50); rc=$?
  if expect_ok "$scene" "$rc" "$out" && [ -n "$OLT_ID" ] && printf '%s' "$out" | grep -q "P-ACC-$GLOBAL_SFX"; then
    sim_pass noc "$scene"
  else
    sim_fail noc "$scene" "置备olt=$OLT_ID 或端口清单缺 P-ACC 标记"
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
