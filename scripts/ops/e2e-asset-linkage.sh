#!/usr/bin/env bash
# e2e-asset-linkage.sh -- 装机联动端到端实测(102 真实部署,禁 mock)。P2-T3/terms.md §1 12 环节。
# 断言: L0 造数基线;S1-S12 环节逐个推进(order.stage/order_stages/四码/认证账号/下发任务);
#       L1 资产 DEPLOYED+绑地址;L2 asset_lifecycles 新增 DEPLOYED 行;L3 scan_logs 有 MATCH;
#       L4 quad_links LINKED;D1-D3 拆机回 IN_STOCK+清地址/IN_STOCK 轨迹行/UNLINKED。
#       拆机链路不可达时输出 "SKIP: [D*] ..." 与原因,不假装通过。
# 造数: acc_ 前缀族隔离(口径同 mainchain-acceptance.sh);收尾 acceptance-cleanup --apply 全量回收
#       + e2e-asset-linkage-residue.sql 逐类残留断言为零 + db-patrol-gate 孤儿门禁,不留孤儿。
# 信号: 每条断言独立输出 "PASS: [编号] ..." / "FAIL: [编号] 期望=.. 实际=.. 上下文=..";
#       收尾汇总 "E2E-ASSET-LINKAGE RESULT: ..." 可 grep。
# 用法: scripts/ops/e2e-asset-linkage.sh [RUNS]   # 缺省 1 轮;幂等,可连续重复执行
# 环境: BASE_URL ADMIN_API_KEY CUSTOMER_ID OFFER_ID CHANNEL_ID MASTER_ID SSH_HOST
#       PROVISION_WAIT SKIP_CLEANUP=1 SKIP_PATROL=1
#       E2E_INJECT_FAIL=1  (A2 演练: 故意把 L1 期望改成 MAINTENANCE,脚本必须变红)
# 依赖: curl python3 ssh(102 免密);鉴权 X-API-Key(test-accounts.json admin key)。
set -u

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
source "$ROOT/scripts/ops/acceptance-lock.sh"
acquire_acceptance_lock || exit 1
trap release_acceptance_lock EXIT

env_or() { local v; v=$(printenv "$1" 2>/dev/null); if [ -n "$v" ]; then echo "$v"; else echo "$2"; fi; }

RUNS="1"; if [ $# -ge 1 ]; then RUNS="$1"; fi
BASE_URL=$(env_or BASE_URL "http://192.168.0.102:28080")
API="$BASE_URL/api/admin/v1"
KEY=$(env_or ADMIN_API_KEY "$(python3 -c "import json;print(json.load(open('$ROOT/.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key'])")")
CUSTOMER_ID=$(env_or CUSTOMER_ID "214")   # 已实名客户(test-accounts.json)
OFFER_ID=$(env_or OFFER_ID "101"); CHANNEL_ID=$(env_or CHANNEL_ID "102")  # PUBLISHED 产品/HALL
MASTER_ID=$(env_or MASTER_ID "7")  # 师傅(区域1 集团): 工单区域强匹配
SSH_HOST=$(env_or SSH_HOST "imeepos@192.168.0.102"); PROVISION_WAIT=$(env_or PROVISION_WAIT "30")
INJECT_FAIL=$(env_or E2E_INJECT_FAIL "0")

# sql: 102 断言/夹具 SQL(姿势同 mainchain-acceptance.sh;stdin 传 SQL 防叠引号)。
sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }

FAIL_REASON=""
# api METHOD PATH [JSON] -> stdout 业务 data;失败置 FAIL_REASON 并返回 1(姿势同 mainchain-acceptance.sh)。
api() {
  local method="$1" path="$2" data="" body out h1="X-API-Key: $KEY" h2="Content-Type: application/json"
  if [ $# -ge 3 ]; then data="$3"; fi
  if [ -n "$data" ]; then
    if ! body=$(curl -sS -m 30 -X "$method" "$API$path" -H "$h1" -H "$h2" -d "$data"); then FAIL_REASON="curl $path"; return 1; fi
  else
    if ! body=$(curl -sS -m 30 -X "$method" "$API$path" -H "$h1" -H "$h2"); then FAIL_REASON="curl $path"; return 1; fi
  fi
  out=$(python3 -c "
import json,sys
try: d=json.loads(sys.argv[1])
except Exception: d={}
print(json.dumps(d.get('data',d)) if d.get('code') in (0,200) else '')" "$body")
  if [ -z "$out" ]; then
    FAIL_REASON="$path -> $(echo "$body" | head -c 200)"
    echo "[api-fail] $FAIL_REASON" >&2
    return 1
  fi
  echo "$out"
}

j() { python3 -c "import json,sys;print(json.loads(sys.argv[1]).get(sys.argv[2],''))" "$1" "$2"; }

# order_field ORDER_NO FIELD -> orders 详情里的字段(stage/status)
order_field() {
  local d; d=$(api GET "/orders/$1") || { echo ""; return 1; }
  python3 -c "import json,sys;print(json.loads(sys.argv[1])['order'].get(sys.argv[2],''))" "$d" "$2"
}

PASS_N=0; FAIL_N=0; SKIP_N=0; FAILED_IDS=""
ok()   { PASS_N=$((PASS_N+1)); echo "PASS: [$1] $2"; }
skip() { SKIP_N=$((SKIP_N+1)); echo "SKIP: [$1] $2"; }
bad() {
  FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"
  echo "FAIL: [$1] $2" >&2
  echo "[e2e-asset-linkage] ASSERTION FAILED id=$1 detail=$2" >&2
}
assert_eq() { # assert_eq ID EXPECTED ACTUAL CONTEXT
  if [ "$2" = "$3" ]; then ok "$1" "$2 ($4)"; else bad "$1" "期望=$2 实际=$3 上下文=$4"; fi
}

SUFFIX=""; ADDR_ID=""; RES_ID=0; TPL_ID=0; BATCH_ID=0; TAG_ID=0; ASSET_ID=0; EPC=""
ORDER_NO=""; ORDER_ID=0; TICKET_NO=""; PORT_ID=0

boot_fixtures() { # 自举: 地址/OLT/模板/端口/批次/标签/资产(acc_ 前缀族)
  local addr res tpl p1 p2 batch tag asset base
  addr=$(api POST /addresses "{\"label\":\"acc_$SUFFIX\",\"name\":\"验收地址-$SUFFIX\"}") || return 1
  ADDR_ID=$(j "$addr" id)
  res=$(api POST /provision/resources "{\"code\":\"OLT-ACC-$SUFFIX\",\"name\":\"验收OLT-$SUFFIX\",\"type\":\"OLT\",\"addressId\":$ADDR_ID,\"legalEntityId\":1}") || return 1
  RES_ID=$(j "$res" id)
  tpl=$(api POST /provision-templates "{\"legalEntityId\":1,\"code\":\"TPL-ACC-$SUFFIX\",\"name\":\"验收模板-$SUFFIX\",\"content\":{\"bandwidth\":\"100M\",\"onuType\":\"Internet\",\"services\":{\"internet\":{\"svlan\":1113,\"cvlan\":1,\"uv\":100,\"scos\":0,\"ccos\":0},\"tr069\":{\"cvlan\":1000,\"uv\":1000,\"scos\":6,\"ccos\":6}}}}") || return 1
  TPL_ID=$(j "$tpl" id)
  echo "UPDATE resources SET nms_oltid='NMS-ACC-$SUFFIX' WHERE id=$RES_ID;" | sql || { FAIL_REASON="sql nms_oltid"; return 1; }
  p1=$(api POST /provision/ports "{\"portCode\":\"P-ACC-$SUFFIX-01\",\"resourceId\":$RES_ID,\"addressId\":$ADDR_ID,\"legalEntityId\":1}") || return 1
  p2=$(api POST /provision/ports "{\"portCode\":\"P-ACC-$SUFFIX-02\",\"resourceId\":$RES_ID,\"addressId\":$ADDR_ID,\"legalEntityId\":1}") || return 1
  batch=$(api POST /provision/asset-batches "{\"code\":\"RK-ACC-$SUFFIX\",\"name\":\"验收批次-$SUFFIX\",\"legalEntityId\":1}") || return 1
  BATCH_ID=$(j "$batch" id)
  tag=$(api POST /provision/tags "{\"tagNo\":\"T-ACC-$SUFFIX\",\"epcCode\":\"EPC-ACC-$SUFFIX\",\"legalEntityId\":1}") || return 1
  TAG_ID=$(j "$tag" id)
  EPC="EPC-ACC-$SUFFIX"
  asset=$(api POST /provision/assets "{\"assetCode\":\"A-ACC-$SUFFIX\",\"batchId\":$BATCH_ID,\"tagId\":$TAG_ID,\"legalEntityId\":1}") || return 1
  ASSET_ID=$(j "$asset" id)
  if [ -z "$ADDR_ID" ] || [ -z "$ASSET_ID" ] || [ "$ASSET_ID" = "None" ]; then
    FAIL_REASON="fixture ids incomplete addr=$ADDR_ID asset=$ASSET_ID"; return 1
  fi
  base=$(echo "SELECT status || '|' || COALESCE(address_id::text,'NULL') FROM assets WHERE id=$ASSET_ID;" | sql | tr -d "[:space:]")
  assert_eq "L0" "IN_STOCK|NULL" "$base" "造数基线 assets(id=$ASSET_ID A-ACC-$SUFFIX)"
}

walk_order() { # 环节1-3: 下单/资源核查/端口预占
  local stage pstatus order
  order=$(api POST /orders "{\"customerId\":$CUSTOMER_ID,\"offerId\":$OFFER_ID,\"addressId\":$ADDR_ID,\"channelId\":$CHANNEL_ID}") || return 1
  ORDER_NO=$(j "$order" orderNo); ORDER_ID=$(j "$order" id)
  stage=$(order_field "$ORDER_NO" stage) || return 1
  assert_eq "S1" "1" "$stage" "下单 orderNo=$ORDER_NO"
  api POST "/orders/$ORDER_NO/check-resource" >/dev/null || return 1
  stage=$(order_field "$ORDER_NO" stage) || return 1
  assert_eq "S2" "2" "$stage" "资源核查"
  api POST "/orders/$ORDER_NO/reserve" >/dev/null || return 1
  stage=$(order_field "$ORDER_NO" stage) || return 1
  assert_eq "S3" "3" "$stage" "端口预占"
  PORT_ID=$(echo "SELECT id FROM ports WHERE order_id=$ORDER_ID AND status='RESERVED' ORDER BY id LIMIT 1;" | sql | tr -d "[:space:]")
  if [ -z "$PORT_ID" ]; then FAIL_REASON="reserved port not found (order $ORDER_ID)"; return 1; fi
  pstatus=$(echo "SELECT status FROM ports WHERE id=$PORT_ID;" | sql | tr -d "[:space:]")
  assert_eq "S3-port" "RESERVED" "$pstatus" "端口 ports.id=$PORT_ID 已锁给本单"
}

# wait_provision_done ORDER_ID: 环节7 下发任务终态 DONE 且最新日志 SUCCESS(轮询 PROVISION_WAIT 秒)。
wait_provision_done() {
  local oid="$1" deadline ts cls id logs latest
  deadline=$(( $(date +%s) + PROVISION_WAIT ))
  while :; do
    ts=$(api GET /provision-tasks) || return 1
    cls=$(python3 -c "
import json,sys
d=json.loads(sys.argv[1])
ts=[t for t in d.get('items',[]) if t.get('orderId')==int(sys.argv[2])]
print('NONE' if not ts else 'PENDING' if any(t.get('status')!='DONE' for t in ts) else 'DONE ' + ' '.join(str(t['id']) for t in ts))" "$ts" "$oid")
    case "$cls" in
      NONE) FAIL_REASON="no provision tasks for order $oid"; return 1 ;;
      PENDING)
        if [ "$(date +%s)" -lt "$deadline" ]; then sleep 2; continue; fi
        FAIL_REASON="provision tasks not DONE within $PROVISION_WAIT s (order $oid)"; return 1 ;;
    esac
    break
  done
  for id in $cls; do
    [ "$id" = "DONE" ] && continue
    logs=$(api GET "/provision-logs?taskId=$id") || return 1
    latest=$(python3 -c "
import json,sys
items=json.loads(sys.argv[1]).get('items',[])
print(items[-1].get('result','') if items else 'NO-LOG')" "$logs")
    case "$latest" in
      SUCCESS*) : ;;
      *) FAIL_REASON="provision task $id latest log=<$latest>"; return 1 ;;
    esac
  done
  return 0
}

walk_auto() { # 环节4-8: 收费触发自动段(applyTag/createUserProfile/preConfigOLT/dispatchOrder)
  local stage ql lohit pool tk
  echo "UPDATE ports SET pon_frame=0,pon_slot=7,pon_port=1,onu_no=NULL WHERE id=$PORT_ID;" | sql || { FAIL_REASON="sql pon dims"; return 1; }
  local lo_id; lo_id=$(echo "SELECT id FROM lo_accounts WHERE customer_id=$CUSTOMER_ID ORDER BY id LIMIT 1;" | sql | tr -d "[:space:]")
  if [ -z "$lo_id" ]; then FAIL_REASON="lo account missing for customer $CUSTOMER_ID"; return 1; fi
  echo "INSERT INTO provision_tasks(task_no,order_id,stage_event,lo_account_id,template_id,status) VALUES('PRV-O$ORDER_ID',$ORDER_ID,'preConfigOLT',$lo_id,$TPL_ID,'PENDING');" | sql || { FAIL_REASON="sql pre provision task"; return 1; }
  api POST "/orders/$ORDER_NO/charge" >/dev/null || return 1
  stage=$(order_field "$ORDER_NO" stage) || return 1
  assert_eq "S4-8" "8" "$stage" "合同收费+自动段(环节5-8)"
  ql=$(echo "SELECT status || '|' || COALESCE(asset_id::text,'0') FROM quad_links WHERE port_id=$PORT_ID ORDER BY id DESC LIMIT 1;" | sql | tr -d "[:space:]")
  assert_eq "S5" "UNLINKED|0" "$ql" "标签预绑定 quad_links(port=$PORT_ID 资产待扫码回填)"
  lohit=$(echo "SELECT COALESCE(offer_id::text,'') || '|' || COALESCE(billing_mode,'POSTPAID') FROM lo_accounts WHERE customer_id=$CUSTOMER_ID ORDER BY id LIMIT 1;" | sql | tr -d "[:space:]")
  assert_eq "S6" "$OFFER_ID|POSTPAID" "$lohit" "创建账号 lo_accounts(customer=$CUSTOMER_ID 生效套餐对齐)"
  if wait_provision_done "$ORDER_ID"; then
    ok "S7" "预下发配置 provision_tasks DONE 且日志 SUCCESS(order=$ORDER_ID)"
  else
    bad "S7" "下发未达终态: $FAIL_REASON"
  fi
  pool=$(api GET /dispatch/pool) || return 1
  TICKET_NO=$(python3 -c "
import json,sys
d=json.loads(sys.argv[1])
print(next((t['ticketNo'] for t in d.get('items',[]) if t.get('orderId')==int(sys.argv[2])), ''))" "$pool" "$ORDER_ID")
  if [ -z "$TICKET_NO" ]; then FAIL_REASON="ticket not in pool (order $ORDER_ID)"; return 1; fi
  api POST "/dispatch/pool/$TICKET_NO/assign" "{\"masterId\":$MASTER_ID}" >/dev/null || return 1
  tk=$(echo "SELECT status || '|' || COALESCE(worker_id::text,'0') FROM dispatch_tickets WHERE ticket_no='$TICKET_NO';" | sql | tr -d "[:space:]")
  assert_eq "S8" "PENDING|$MASTER_ID" "$tk" "派单 ticket=$TICKET_NO 指派师傅=$MASTER_ID"
}

walk_scan() { # 环节9 扫码绑定 + 资产联动断言(本脚本核心)
  local stage want arow lc sl ql
  api POST "/tickets/$TICKET_NO/scan-bind" "{\"epc\":\"$EPC\"}" >/dev/null || return 1
  stage=$(order_field "$ORDER_NO" stage) || return 1
  assert_eq "S9" "9" "$stage" "扫码绑定 result=MATCH(epc=$EPC)"
  want="DEPLOYED"
  if [ "$INJECT_FAIL" = "1" ]; then want="MAINTENANCE"; fi
  arow=$(echo "SELECT status || '|' || COALESCE(address_id::text,'NULL') FROM assets WHERE id=$ASSET_ID;" | sql | tr -d "[:space:]")
  assert_eq "L1" "$want|$ADDR_ID" "$arow" "资产联动 assets(id=$ASSET_ID A-ACC-$SUFFIX) DEPLOYED+绑地址"
  lc=$(echo "SELECT count(*) FROM asset_lifecycles WHERE asset_id=$ASSET_ID AND status='DEPLOYED' AND address_id=$ADDR_ID;" | sql | tr -d "[:space:]")
  assert_eq "L2" "1" "$lc" "asset_lifecycles 新增 DEPLOYED 行(asset=$ASSET_ID)"
  sl=$(echo "SELECT count(*) FROM scan_logs WHERE order_id=$ORDER_ID AND result='MATCH';" | sql | tr -d "[:space:]")
  assert_eq "L3" "1" "$sl" "scan_logs MATCH(order=$ORDER_NO)"
  ql=$(echo "SELECT status FROM quad_links WHERE port_id=$PORT_ID ORDER BY id DESC LIMIT 1;" | sql | tr -d "[:space:]")
  assert_eq "L4" "LINKED" "$ql" "quad_links(port=$PORT_ID) 一致性对账态"
}

walk_finish() { # 环节10-12: 激活触发自动段(activateUser/notifyActivation/updateMap)
  local deadline stage status s10 s11 s12
  api POST "/tickets/$TICKET_NO/activate" >/dev/null || return 1
  deadline=$(( $(date +%s) + 15 ))
  while :; do
    stage=$(order_field "$ORDER_NO" stage) || return 1
    status=$(order_field "$ORDER_NO" status) || return 1
    if [ "$stage" = "12" ] && [ "$status" = "DONE" ]; then break; fi
    if [ "$(date +%s)" -ge "$deadline" ]; then FAIL_REASON="order not 12/DONE within 15s (now $stage/$status)"; return 1; fi
    sleep 2
  done
  s10=$(echo "SELECT count(*) FROM order_stages WHERE order_id=$ORDER_ID AND stage=10;" | sql | tr -d "[:space:]")
  s11=$(echo "SELECT count(*) FROM order_stages WHERE order_id=$ORDER_ID AND stage=11;" | sql | tr -d "[:space:]")
  s12=$(echo "SELECT count(*) FROM order_stages WHERE order_id=$ORDER_ID AND stage=12;" | sql | tr -d "[:space:]")
  assert_eq "S10" "1" "$s10" "激活 activateUser(order_stages 行)"
  assert_eq "S11" "1" "$s11" "激活回调 notifyActivation(order_stages 行)"
  assert_eq "S12" "1" "$s12" "更新GIS updateMap(order_stages 行;订单 12/DONE)"
}

walk_dismantle() { # 拆机联动(可达则断言 D1-D3;不可达 SKIP+原因,不假装通过)
  local guard drow dlc dql
  guard=$(echo "SELECT port_id || '|' || status FROM quad_links WHERE customer_id=$CUSTOMER_ID ORDER BY id DESC LIMIT 1;" | sql | tr -d "[:space:]")
  if [ "$guard" != "$PORT_ID|LINKED" ]; then
    skip "D1-D3" "拆机链路不可达: customer=$CUSTOMER_ID 最新四码链路=<$guard> 非本单(port=$PORT_ID LINKED);拆机按客户定位链路,防误解绑跳过"
    return 0
  fi
  api POST /dismantles "{\"dismantleNo\":\"CJ-ACC-$SUFFIX\",\"orderId\":$ORDER_ID,\"legalEntityId\":1,\"legalEntityName\":\"验收主体\",\"assetId\":$ASSET_ID,\"portId\":$PORT_ID,\"status\":\"PENDING\"}" >/dev/null \
    || echo "[e2e-asset-linkage] WARN 拆机单台账创建失败(不阻断资产联动断言): $FAIL_REASON" >&2
  api POST "/tickets/$TICKET_NO/dismantle/scan" "{\"epc\":\"$EPC\"}" >/dev/null || { FAIL_REASON="dismantle scan rejected: $FAIL_REASON"; return 1; }
  drow=$(echo "SELECT status || '|' || COALESCE(address_id::text,'NULL') FROM assets WHERE id=$ASSET_ID;" | sql | tr -d "[:space:]")
  assert_eq "D1" "IN_STOCK|NULL" "$drow" "拆机联动 assets(id=$ASSET_ID) 回库存+清地址"
  dlc=$(echo "SELECT count(*) FROM asset_lifecycles WHERE asset_id=$ASSET_ID AND status='IN_STOCK';" | sql | tr -d "[:space:]")
  assert_eq "D2" "1" "$dlc" "asset_lifecycles 新增 IN_STOCK 行(asset=$ASSET_ID)"
  dql=$(echo "SELECT status FROM quad_links WHERE port_id=$PORT_ID ORDER BY id DESC LIMIT 1;" | sql | tr -d "[:space:]")
  assert_eq "D3" "UNLINKED" "$dql" "quad_links 解绑(port=$PORT_ID)"
}

residue_gate() { # A3: 收尾后库内 acc_ 前缀残留必须为零(SQL 逐类断言,语句在伴生 .sql)
  local q rc cls n
  echo "收尾: 前缀残留断言(acceptance-cleanup --apply 后应为零)"
  rc=0
  q=$(sql < "$ROOT/scripts/ops/e2e-asset-linkage-residue.sql")
  if [ -z "$q" ]; then bad "RES-db" "残留查询失败(ssh/psql 不可达),不能假装干净"; return 1; fi
  while IFS="|" read -r cls n; do
    [ -z "$cls" ] && continue
    if [ "$n" = "0" ]; then ok "RES-$cls" "残留=0"; else bad "RES-$cls" "收尾残留 count=$n(应为零,查 acceptance-cleanup 口径)"; rc=1; fi
  done <<EOF
$q
EOF
  return $rc
}

echo "装机联动端到端实测: $RUNS 轮 @ $BASE_URL (customer=$CUSTOMER_ID offer=$OFFER_ID master=$MASTER_ID)"
if [ "$INJECT_FAIL" = "1" ]; then echo "  [inject] E2E_INJECT_FAIL=1: L1 资产状态期望已故意改错,脚本必须变红(A2 演练)"; fi
# 直营风控临时关停(同客户短时高频下单触发 phoneCap 42300;收尾恢复,口径同 mainchain-acceptance.sh)
RISK_OFF=0
if curl -sS -m 10 "$API/params/risk.direct.enabled" -H "X-API-Key: $KEY" 2>/dev/null | grep -q '"value":"true"'; then
  curl -sS -m 10 -X PUT "$API/params/risk.direct.enabled" -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d '{"value":"false"}' >/dev/null
  RISK_OFF=1; echo "  [risk] 已临时关停直营风控(收尾恢复)"
fi
r=0
while [ "$r" -lt "$RUNS" ]; do
  r=$((r+1))
  t0=$(date +%s)
  SUFFIX="$(date +%s)-$r-$RANDOM"
  echo "== run $r suffix=$SUFFIX =="
  if boot_fixtures "$SUFFIX" && walk_order && walk_auto && walk_scan && walk_finish && walk_dismantle; then
    echo "  run $r 断言完成 ($(( $(date +%s) - t0 ))s)"
  else
    bad "RUN$r" "流程中断: $FAIL_REASON(造数 suffix=$SUFFIX 将由收尾清理回收)"
  fi
done

rc=0
if [ "$(env_or SKIP_CLEANUP 0)" != "1" ]; then
  echo "收尾: 造数自清理(acceptance-cleanup --apply,备份+单事务)"
  "$ROOT/scripts/ops/acceptance-cleanup.sh" --apply || { bad "CLEANUP" "acceptance-cleanup --apply 失败"; rc=1; }
fi
residue_gate || rc=1
if [ "$(env_or SKIP_PATROL 0)" != "1" ]; then
  echo "收尾: 孤儿巡检门禁(db-patrol-gate)"
  "$ROOT/scripts/ops/db-patrol-gate.sh" || { bad "PATROL" "孤儿巡检超阈值"; rc=1; }
fi
if [ "$RISK_OFF" = "1" ]; then
  curl -sS -m 10 -X PUT "$API/params/risk.direct.enabled" -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d '{"value":"true"}' >/dev/null
  echo "  [risk] 直营风控已恢复"
fi
echo "断言明细: PASS=$PASS_N FAIL=$FAIL_N SKIP=$SKIP_N 失败编号=[$FAILED_IDS ]"
if [ "$FAIL_N" -eq 0 ] && [ "$rc" -eq 0 ]; then
  echo "E2E-ASSET-LINKAGE RESULT: PASS runs=$RUNS pass=$PASS_N skip=$SKIP_N"
  exit 0
fi
echo "E2E-ASSET-LINKAGE RESULT: FAIL runs=$RUNS pass=$PASS_N fail=$FAIL_N skip=$SKIP_N failed=[$FAILED_IDS ]"
exit 1
