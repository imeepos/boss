#!/usr/bin/env bash
# 主链路验收(实名→订购→预占→收费→派单→扫码→激活→GIS):对真实 102 环境逐环节打点。
# 每轮自举资源(地址/分光器/端口/标签/资产),可无限重复;RUNS 控制轮数,度量成功率。
# 用法: scripts/ops/mainchain-acceptance.sh [RUNS]   (默认 5)
# 环境: BASE_URL / ADMIN_API_KEY 可覆盖;缺省 102 + test-accounts.json admin key。
set -u

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

one_run() {
  local run="$1" suffix
  suffix="$(date +%s)%03d$RANDOM"; suffix=$(printf "$suffix" "$run")
  FAIL_REASON=""
  # 自举资源:地址 + 分光器 + 2端口 + 标签/资产(扫码比对源)
  local addr res tag asset
  addr=$(api POST /addresses "{\"label\":\"acc_$suffix\",\"name\":\"验收地址-$suffix\"}") || return 1
  local addr_id; addr_id=$(j "$addr" id)
  res=$(api POST /provision/resources "{\"code\":\"SPL-ACC-$suffix\",\"name\":\"验收分光器$run\",\"type\":\"SPLITTER\",\"addressId\":$addr_id,\"legalEntityId\":1}") || return 1
  local res_id; res_id=$(j "$res" id)
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
exit 1
