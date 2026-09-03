#!/usr/bin/env bash
# 主链路验收(实名→订购→预占→收费→派单→扫码→激活→GIS):对真实 102 环境逐环节打点。
# 每轮自举资源(地址/分光器/端口/标签/资产),可无限重复;RUNS 控制轮数,度量成功率。
# 用法: scripts/ops/mainchain-acceptance.sh [RUNS]   (默认 5)
# 环境: BASE_URL / ADMIN_API_KEY 可覆盖;缺省 102 + test-accounts.json admin key。
set -u

source "$(dirname "$0")/acceptance-lock.sh"
acquire_acceptance_lock || exit 1
trap release_acceptance_lock EXIT

RUNS="${1:-5}"
BASE_URL="${BASE_URL:-http://192.168.0.102:28080}"
API="${BASE_URL}/api/admin/v1"
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
KEY="${ADMIN_API_KEY:-$(python3 -c "import json,sys;print(json.load(open('$ROOT/.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key'])")}"
CUSTOMER_ID="${CUSTOMER_ID:-214}"   # 已实名客户(test-accounts.json)
OFFER_ID="${OFFER_ID:-101}"         # PUBLISHED 产品
CHANNEL_ID="${CHANNEL_ID:-102}"     # HALL 营业厅
MASTER_ID="${MASTER_ID:-7}"         # 师傅 杨明明(区域1 集团):geo-unify 派单强匹配
# 工单区域=师傅区域,验收地址不带 region 时工单解析为根区域"集团",师傅须同在区域1
# (6 号王测试在区域4 马尼拉必 40900 region mismatch);跨区域场景用 MASTER_ID 覆盖。

SSH_HOST="${SSH_HOST:-imeepos@192.168.0.102}"
# sql: 102 夹具 SQL(仅限无管理面的字段:nms_oltid/PON 三维/预置任务,姿势同 verify-tl1-e2e.sh)。
sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }

OK=0; FAIL=0; FAILED_RUNS=()

# api METHOD PATH [JSON] -> stdout 业务 data;失败时输出空串并记 FAIL_REASON。
FAIL_REASON=""
api() {
  local method="$1" path="$2" data="${3:-}"
  local args=(-sS -m 30 -X "$method" "$API$path" -H "X-API-Key: $KEY" -H "Content-Type: application/json")
  [ -n "$data" ] && args+=(-d "$data")
  local body
  body=$(curl "${args[@]}") || { FAIL_REASON="curl $path"; return 1; }
  local out
  out=$(python3 -c "
import json,sys
try: d=json.loads('''$body''')
except Exception: print(''); sys.exit(0)
if d.get('code') not in (0,200): print(''); sys.exit(0)
print(json.dumps(d.get('data',d)))")
  if [ -z "$out" ]; then FAIL_REASON="$path -> $(echo "$body" | head -c 200)"; echo "[api-fail] $FAIL_REASON" >&2; return 1; fi
  echo "$out"
}

# j JSON FIELD -> 取字段值
j() { python3 -c "import json,sys;print(json.loads(sys.argv[1]).get(sys.argv[2],''))" "$1" "$2"; }

# 终态断言(2026-09-04 D1 补全):stage=12/DONE 不再是唯一终态口径——认证账号与下发结果
# 必须同轮验证,任一失败判本轮 FAIL(此前「开户/拨号上网」两环在 102 零自动化覆盖)。

# assert_lo_account_matches_order: 该订单客户的 LO 账号已建立,生效套餐/付费模式与订单一致。
# 依据: 环节6 创建账号(pg_workflow CreateUserProfile:客户 1:1 账号,改套餐即对齐 offer)。
assert_lo_account_matches_order() {
  local detail="$1" order_no="$2" lo hit loid got_offer got_mode
  local want_offer want_mode cust_id
  want_offer=$(python3 -c "import json;print(json.loads('''$detail''')['order'].get('offerId',''))")
  want_mode=$(python3 -c "import json;print(json.loads('''$detail''')['order'].get('billingMode') or 'POSTPAID')")
  cust_id=$(python3 -c "import json;print(json.loads('''$detail''')['order'].get('customerId',''))")
  lo=$(api GET "/lo-accounts?pageSize=200") || return 1
  hit=$(python3 -c "
import json
d=json.loads('''$lo''')
for i in d.get('items',[]):
    if i.get('customerId')==$cust_id:
        print(i.get('loid',''), i.get('offerId',''), i.get('billingMode') or 'POSTPAID'); break")
  loid=$(echo "$hit" | awk '{print $1}'); got_offer=$(echo "$hit" | awk '{print $2}'); got_mode=$(echo "$hit" | awk '{print $3}')
  if [ -z "$loid" ]; then
    FAIL_REASON="lo-account missing for customer $cust_id (order $order_no)"
    echo "[auth-assert] FAIL order=$order_no: $FAIL_REASON" >&2; return 1
  fi
  if [ "$got_offer" != "$want_offer" ] || [ "$got_mode" != "$want_mode" ]; then
    FAIL_REASON="lo-account mismatch: got(offer=$got_offer mode=$got_mode) want(offer=$want_offer mode=$want_mode)"
    echo "[auth-assert] FAIL order=$order_no loid=$loid: $FAIL_REASON" >&2; return 1
  fi
  echo "[auth-assert] PASS order=$order_no loid=$loid 生效套餐一致=$got_offer 付费模式一致=$got_mode" >&2
}

# assert_provision_success: 该订单全部下发任务终态 DONE,且每任务最新下发日志终态 SUCCESS。
# 依据: 环节7/10/11 入队 provision_tasks,provisioner 异步执行,轮询等待(PROVISION_WAIT 缺省 30s)。
assert_provision_success() {
  local order_id="$1" order_no="$2" deadline=$(( $(date +%s) + ${PROVISION_WAIT:-30} )) ts cls id logs latest
  while :; do
    ts=$(api GET /provision-tasks) || return 1
    cls=$(python3 -c "
import json
d=json.loads('''$ts''')
ts=[t for t in d.get('items',[]) if t.get('orderId')==$order_id]
if not ts: print('NONE')
elif any(t.get('status')!='DONE' for t in ts): print('PENDING')
else: print('DONE ' + ' '.join(str(t['id']) for t in ts))")
    case "$cls" in
      NONE)
        FAIL_REASON="no provision tasks for order $order_id"
        echo "[provision-assert] FAIL order=$order_no: $FAIL_REASON" >&2; return 1 ;;
      PENDING)
        if [ "$(date +%s)" -lt "$deadline" ]; then sleep 2; continue; fi
        FAIL_REASON="provision tasks not DONE within ${PROVISION_WAIT:-30}s (order $order_id)"
        echo "[provision-assert] FAIL order=$order_no: $FAIL_REASON" >&2; return 1 ;;
    esac
    break
  done
  local n=0
  for id in $cls; do
    [ "$id" = "DONE" ] && continue
    n=$((n+1))
    logs=$(api GET "/provision-logs?taskId=$id") || return 1
    latest=$(python3 -c "
import json
d=json.loads('''$logs''')
items=d.get('items',[])
print(items[-1].get('result','') if items else 'NO-LOG')")
    case "$latest" in
      SUCCESS*) : ;;
      *) FAIL_REASON="provision task $id latest log=<$latest>"
         echo "[provision-assert] FAIL order=$order_no task=$id latest-log=<$latest>" >&2; return 1 ;;
    esac
  done
  echo "[provision-assert] PASS order=$order_no 下发任务=$n 全部日志 SUCCESS" >&2
}

one_run() {
  local run="$1" suffix
  suffix="$(date +%s)%03d$RANDOM"; suffix=$(printf "$suffix" "$run")
  FAIL_REASON=""
  # 自举资源(TL1 驱动就绪):地址 + OLT + 2端口 + TL1 模板 + 标签/资产(扫码比对源)。
  # 102 已切 TL1 驱动(BOSS_PROVISION_DRIVER=tl1):裸 SPLITTER/无 PON 定位端口会让
  # 环节7 任务 RESOLVE FAILED(port missing PON positioning),订单却照样 stage=12——
  # 正是本脚本新增下发断言要暴露的盲区,故夹具须带 nms_oltid/PON 三维/TL1 内容模板。
  local addr res tag asset tpl
  addr=$(api POST /addresses "{\"label\":\"acc_$suffix\",\"name\":\"验收地址-$suffix\"}") || return 1
  local addr_id; addr_id=$(j "$addr" id)
  res=$(api POST /provision/resources "{\"code\":\"OLT-ACC-$suffix\",\"name\":\"验收OLT$run\",\"type\":\"OLT\",\"addressId\":$addr_id,\"legalEntityId\":1}") || return 1
  local res_id; res_id=$(j "$res" id)
  tpl=$(api POST /provision-templates "{\"legalEntityId\":1,\"code\":\"TPL-ACC-$suffix\",\"name\":\"验收模板$run\",\"content\":{\"bandwidth\":\"100M\",\"onuType\":\"Internet\",\"services\":{\"internet\":{\"svlan\":1113,\"cvlan\":1,\"uv\":100,\"scos\":0,\"ccos\":0},\"tr069\":{\"cvlan\":1000,\"uv\":1000,\"scos\":6,\"ccos\":6}}}}") || return 1
  local tpl_id; tpl_id=$(j "$tpl" id)
  echo "UPDATE resources SET nms_oltid='NMS-ACC-$suffix' WHERE id=$res_id;" | sql || return 1
  local pids=()
  for i in 1 2; do
    local p; p=$(api POST /provision/ports "{\"portCode\":\"P-ACC-$suffix-0$i\",\"resourceId\":$res_id,\"addressId\":$addr_id,\"legalEntityId\":1}") || return 1
    pids+=("$(j "$p" id)")
  done
  local batch; batch=$(api POST /provision/asset-batches "{\"code\":\"RK-ACC-$suffix\",\"name\":\"验收批次$run\",\"legalEntityId\":1}") || return 1
  tag=$(api POST /provision/tags "{\"tagNo\":\"T-ACC-$suffix\",\"epcCode\":\"EPC-ACC-$suffix\",\"legalEntityId\":1}") || return 1
  asset=$(api POST /provision/assets "{\"assetCode\":\"A-ACC-$suffix\",\"batchId\":$(j "$batch" id),\"tagId\":$(j "$tag" id),\"legalEntityId\":1}") || return 1
  # 主链路 12 环节(实名前置:客户 $CUSTOMER_ID 已 VERIFIED,见 test-accounts.json)
  local order; order=$(api POST /orders "{\"customerId\":$CUSTOMER_ID,\"offerId\":$OFFER_ID,\"addressId\":$addr_id,\"channelId\":$CHANNEL_ID}") || return 1
  local order_no; order_no=$(j "$order" orderNo)
  api POST "/orders/$order_no/check-resource" >/dev/null || return 1
  api POST "/orders/$order_no/reserve" >/dev/null || return 1
  # 预置 TL1 下发(姿势同 verify-tl1-e2e.sh):订单预占端口回填 PON 三维;预建 PENDING 任务
  # (task_no=PRV-O<oid>,环节7 CreateTask 同号幂等复用)保证 TL1 就绪模板必达设备。
  local order_id port_id lo_id
  order_id=$(j "$order" id)
  port_id=$(echo "SELECT id FROM ports WHERE order_id=$order_id AND status='RESERVED' ORDER BY id LIMIT 1;" | sql | tr -d '[:space:]')
  [ -n "$port_id" ] || { FAIL_REASON="reserved port not found"; return 1; }
  echo "UPDATE ports SET pon_frame=0,pon_slot=7,pon_port=$run,onu_no=NULL WHERE id=$port_id;" | sql || return 1
  lo_id=$(echo "SELECT id FROM lo_accounts WHERE customer_id=$CUSTOMER_ID ORDER BY id LIMIT 1;" | sql | tr -d '[:space:]')
  [ -n "$lo_id" ] || { FAIL_REASON="lo account missing for customer $CUSTOMER_ID"; return 1; }
  echo "INSERT INTO provision_tasks(task_no,order_id,stage_event,lo_account_id,template_id,status) VALUES('PRV-O$order_id',$order_id,'preConfigOLT',$lo_id,$tpl_id,'PENDING');" | sql || return 1
  api POST "/orders/$order_no/charge" >/dev/null || return 1
  local pool; pool=$(api GET /dispatch/pool) || return 1
  local ticket_no; ticket_no=$(python3 -c "
import json,sys
d=json.loads('''$pool''')
for t in d.get('items',[]):
    if t.get('orderId')==$(j "$order" id): print(t['ticketNo']); break")
  [ -n "$ticket_no" ] || { FAIL_REASON="ticket not in pool"; return 1; }
  api POST "/dispatch/pool/$ticket_no/assign" "{\"masterId\":$MASTER_ID}" >/dev/null || return 1
  api POST "/tickets/$ticket_no/scan-bind" "{\"epc\":\"EPC-ACC-$suffix\"}" >/dev/null || return 1
  api POST "/tickets/$ticket_no/activate" >/dev/null || return 1
  # 终态断言:stage=12 / DONE
  local detail; detail=$(api GET "/orders/$order_no") || return 1
  local stage status
  stage=$(python3 -c "import json;d=json.loads('''$detail''');print(d['order']['stage'])")
  status=$(python3 -c "import json;d=json.loads('''$detail''');print(d['order']['status'])")
  [ "$stage" = "12" ] && [ "$status" = "DONE" ] || { FAIL_REASON="final $stage/$status"; return 1; }
  assert_lo_account_matches_order "$detail" "$order_no" || return 1
  assert_provision_success "$(j "$order" id)" "$order_no" || return 1
  echo "$order_no"
}

echo "主链路验收: $RUNS 轮 @ $BASE_URL (customer=$CUSTOMER_ID)"
# 直营风控共存:验收前临时关停 risk.direct.enabled,结束后恢复原值。
# 多轮同客户短时高频下单会被 phoneCap 拦截(42300),属正常风控但阻碍验收度量。
RISK_OFF=0
if curl -sS -m 10 "$API/params/risk.direct.enabled" -H "X-API-Key: $KEY" 2>/dev/null | grep -q '"value":"true"'; then
  curl -sS -m 10 -X PUT "$API/params/risk.direct.enabled" -H "X-API-Key: $KEY" \
    -H "Content-Type: application/json" -d '{"value":"false"}' >/dev/null
  RISK_OFF=1; echo "  [risk] 已临时关停直营风控(验收后恢复)"
fi
start=$(date +%s)
for r in $(seq 1 "$RUNS"); do
  t0=$(date +%s)
  if out=$(one_run "$r"); then
    OK=$((OK+1)); echo "  run $r PASS $(echo "$out" | tail -1) ($(( $(date +%s)-t0 ))s)"
  else
    FAIL=$((FAIL+1)); FAILED_RUNS+=("$r"); echo "  run $r FAIL: $FAIL_REASON"
  fi
done
dur=$(( $(date +%s)-start ))
total=$((OK+FAIL)); rate=$(python3 -c "print(f'{$OK/$total*100:.1f}' if $total else '0.0')")
echo "结果: $OK/$total 成功率 $rate%  总耗时 ${dur}s"

# 收尾自清理(2026-08-29 固化):验收造数不过夜——按 acc_ 标记回收本轮及历史造数
# (备份+单事务,见 acceptance-cleanup.sh);SKIP_CLEANUP=1 可跳过。
rc=0
if [ "${SKIP_CLEANUP:-0}" != "1" ]; then
  echo "收尾: 造数自清理"
  "$ROOT/scripts/ops/acceptance-cleanup.sh" --apply || rc=1
fi
# 巡检门禁:任一孤儿类 >0 即失败,防造数泄漏无人察觉(2026-08-25 审计 §五.2)。
echo "收尾: 孤儿巡检门禁"
"$ROOT/scripts/ops/db-patrol-gate.sh" || rc=1

# 恢复直营风控(若验收前被本脚本关停)。
if [ "$RISK_OFF" = "1" ]; then
  curl -sS -m 10 -X PUT "$API/params/risk.direct.enabled" -H "X-API-Key: $KEY" \
    -H "Content-Type: application/json" -d '{"value":"true"}' >/dev/null
  echo "  [risk] 直营风控已恢复"
fi

[ "$FAIL" -eq 0 ] && [ "$rc" -eq 0 ] && exit 0
release_acceptance_lock
exit 1
