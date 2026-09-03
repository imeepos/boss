#!/usr/bin/env bash
# 拨号上网 E2E 验收(D2,2026-09-04):对 102 RADIUS(UDP 1812) 用新开通订单的 LOID 发起真实 Access-Request。
# 三段断言: 1) 在服 Access-Accept 且授权带宽(FramedPool)=订单套餐带宽;
#           2) 停服(既有管理接口 /arrears/:customerId/stop,禁止直改库)后 Access-Reject;
#           3) 复服(/arrears/:customerId/resume)后再次 Access-Accept。
# 造数: acc_ 前缀体系(地址/分光器/端口/标签/资产/订单),与 mainchain-acceptance.sh 同一标记体系,
#       收尾复用 acceptance-cleanup.sh --apply + db-patrol-gate.sh 门禁,可无限重复执行。
# 共享密钥: 运行时经 SSH 从 102 部署配置读取(boss-aaa 容器环境 BOSS_AAA_SECRET),不硬编码进仓库。
# RADIUS 客户端: 仓库内最小实现 scripts/ops/radius_probe.py(零依赖;契约对齐 aaa/radius/handler.go:
#               服务端仅按 LOID 鉴权,Access-Accept 携带 FramedPool=套餐带宽)。
# 负向自证: --negative 用错误共享密钥走同一在服断言,预期该轮 FAIL 退出非 0(证明断言可判死,非恒绿)。
# 用法: scripts/ops/dial-acceptance.sh [RUNS]        # 默认 1
#       scripts/ops/dial-acceptance.sh --negative    # 错误密钥自证,断言判死时退出码 1
# 环境: BASE_URL/ADMIN_API_KEY/SSH_HOST/RADIUS_ADDR/OFFER_ID/CUSTOMER_ID/SKIP_CLEANUP 可覆盖。
set -u

NEGATIVE=0; RUNS=1
for a in "$@"; do
  case "$a" in
    --negative) NEGATIVE=1 ;;
    *) RUNS="$a" ;;
  esac
done
BASE_URL="${BASE_URL:-http://192.168.0.102:28080}"
API="${BASE_URL}/api/admin/v1"
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
KEY="${ADMIN_API_KEY:-$(python3 -c "import json,sys;print(json.load(open('$ROOT/.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key'])")}"
CUSTOMER_ID="${CUSTOMER_ID:-214}"   # 已实名客户(test-accounts.json)
OFFER_ID="${OFFER_ID:-101}"         # PUBLISHED 产品(带宽 100M)
CHANNEL_ID="${CHANNEL_ID:-102}"
MASTER_ID="${MASTER_ID:-7}"         # 师傅在区域1,验收地址不带 region 时工单解析为根区域
SSH_HOST="${SSH_HOST:-imeepos@192.168.0.102}"
RADIUS_ADDR="${RADIUS_ADDR:-192.168.0.102:1812}"
AAA_CONTAINER="${AAA_CONTAINER:-boss-aaa}"
PG="docker exec -i boss-infra-postgres-1 psql -U boss -d boss"
PROBE="$ROOT/scripts/ops/radius_probe.py"

# sql: 102 夹具 SQL(仅限无管理面的字段:nms_oltid/PON 三维/预置任务,姿势同 verify-tl1-e2e.sh)。
sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }

OK=0; FAIL=0; FAILED_RUNS=(); FAIL_REASON=""

# api METHOD PATH [JSON] -> stdout 业务 data;失败时空串 + FAIL_REASON。
api() {
  local method="$1" path="$2" data="${3:-}"
  local args=(-sS -m 30 -X "$method" "$API$path" -H "X-API-Key: $KEY" -H "Content-Type: application/json")
  [ -n "$data" ] && args+=(-d "$data")
  local body out
  body=$(curl "${args[@]}") || { FAIL_REASON="curl $path"; return 1; }
  out=$(python3 -c "
import json,sys
try: d=json.loads('''$body''')
except Exception: print(''); sys.exit(0)
if d.get('code') not in (0,200): print(''); sys.exit(0)
print(json.dumps(d.get('data',d)))")
  if [ -z "$out" ]; then FAIL_REASON="$path -> $(echo "$body" | head -c 200)"; echo "[api-fail] $FAIL_REASON" >&2; return 1; fi
  echo "$out"
}

j() { python3 -c "import json,sys;print(json.loads(sys.argv[1]).get(sys.argv[2],''))" "$1" "$2"; }

# read_secret: 从 102 部署配置读 RADIUS 共享密钥(boss-aaa 容器 env),只入内存不落盘不入仓库。
read_secret() {
  ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec $AAA_CONTAINER printenv BOSS_AAA_SECRET" 2>/dev/null
}

# resolve_loid: 按客户读其 1:1 认证账号 LOID(只读 SQL;写路径一律走管理接口)。
resolve_loid() {
  printf 'SELECT loid FROM lo_accounts WHERE customer_id=%s ORDER BY id LIMIT 1;\n' "$1" \
    | ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "$PG -tA" 2>/dev/null
}

# expected_bandwidth: 订单套餐的权威带宽(product_offers.bandwidth)。
expected_bandwidth() {
  local ps
  ps=$(api GET /products) || return 1
  python3 -c "
import json
d=json.loads('''$ps''')
for p in d.get('items',[]):
    if p.get('id')==$1: print(p.get('bandwidth','')); break"
}

# dial: 向 RADIUS 发真实 Access-Request;stdout 为探针单行结果,退出码 0=收到合法应答。
dial() {
  python3 "$PROBE" --server "$RADIUS_ADDR" --secret "$1" --loid "$2"
}

dial_field() { echo "$1" | sed -n "s/.*$2=\([^ ]*\).*/\1/p"; }

# expect_dial PHASE WANT_CODE WANT_BW PROBE_OUT: 断言应答码(与 Accept 时的带宽)。
expect_dial() {
  local code bw
  code=$(dial_field "$4" code); bw=$(dial_field "$4" bandwidth)
  if [ "$code" != "$2" ]; then
    FAIL_REASON="$1 expect=$2 got=${code:-NO-REPLY}"; return 1
  fi
  if [ "$2" = "Access-Accept" ] && [ "$bw" != "$3" ]; then
    FAIL_REASON="$1 bandwidth got=$bw want=$3"; return 1
  fi
  echo "[dial-acceptance] PASS $1: $code bandwidth=${bw:-_}"
}
# one_run RUN -> stdout 末行=订单号;进度/断言行走 stderr。返回非 0 判本轮 FAIL。
one_run() {
  local run="$1" suffix
  suffix="$(date +%s)%03d$RANDOM"; suffix=$(printf "$suffix" "$run")
  FAIL_REASON=""
  # 自举资源(TL1 驱动就绪,与 mainchain-acceptance.sh 同一 acc_ 标记体系):
  # 地址 + OLT(nms_oltid) + 2端口(预留后回填 PON 三维) + TL1 内容模板 + 标签/资产。
  # 102 已切 TL1 驱动:无 PON 定位夹具会让环节7 任务 RESOLVE FAILED 且订单照样 DONE。
  local addr res tag asset tpl
  addr=$(api POST /addresses "{\"label\":\"acc_$suffix\",\"name\":\"验收地址-$suffix\"}") || return 1
  local addr_id; addr_id=$(j "$addr" id)
  res=$(api POST /provision/resources "{\"code\":\"OLT-ACC-$suffix\",\"name\":\"拨号验收OLT$run\",\"type\":\"OLT\",\"addressId\":$addr_id,\"legalEntityId\":1}") || return 1
  tpl=$(api POST /provision-templates "{\"legalEntityId\":1,\"code\":\"TPL-ACC-$suffix\",\"name\":\"拨号验收模板$run\",\"content\":{\"bandwidth\":\"100M\",\"onuType\":\"Internet\",\"services\":{\"internet\":{\"svlan\":1113,\"cvlan\":1,\"uv\":100,\"scos\":0,\"ccos\":0},\"tr069\":{\"cvlan\":1000,\"uv\":1000,\"scos\":6,\"ccos\":6}}}}") || return 1
  local tpl_id; tpl_id=$(j "$tpl" id)
  echo "UPDATE resources SET nms_oltid='NMS-ACC-$suffix' WHERE id=$(j "$res" id);" | sql || return 1
  local res_id; res_id=$(j "$res" id)
  local res_id; res_id=$(j "$res" id)
  local pids=()
  for i in 1 2; do
    local p; p=$(api POST /provision/ports "{\"portCode\":\"P-ACC-$suffix-0$i\",\"resourceId\":$res_id,\"addressId\":$addr_id,\"legalEntityId\":1}") || return 1
    pids+=("$(j "$p" id)")
  done
  local batch; batch=$(api POST /provision/asset-batches "{\"code\":\"RK-ACC-$suffix\",\"name\":\"拨号验收批次$run\",\"legalEntityId\":1}") || return 1
  tag=$(api POST /provision/tags "{\"tagNo\":\"T-ACC-$suffix\",\"epcCode\":\"EPC-ACC-$suffix\",\"legalEntityId\":1}") || return 1
  asset=$(api POST /provision/assets "{\"assetCode\":\"A-ACC-$suffix\",\"batchId\":$(j "$batch" id),\"tagId\":$(j "$tag" id),\"legalEntityId\":1}") || return 1
  # 主链路 12 环节(同 mainchain-acceptance.sh:客户已实名,风控已由入口临时关停)
  local order order_no
  order=$(api POST /orders "{\"customerId\":$CUSTOMER_ID,\"offerId\":$OFFER_ID,\"addressId\":$addr_id,\"channelId\":$CHANNEL_ID}") || return 1
  order_no=$(j "$order" orderNo)
  api POST "/orders/$order_no/check-resource" >/dev/null || return 1
  api POST "/orders/$order_no/reserve" >/dev/null || return 1
  # 预置 TL1 下发(姿势同 verify-tl1-e2e.sh):预占端口回填 PON 三维;预建 PENDING 任务
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
  local pool ticket_no
  pool=$(api GET /dispatch/pool) || return 1
  ticket_no=$(python3 -c "
import json,sys
d=json.loads('''$pool''')
for t in d.get('items',[]):
    if t.get('orderId')==$(j "$order" id): print(t['ticketNo']); break")
  [ -n "$ticket_no" ] || { FAIL_REASON="ticket not in pool"; return 1; }
  api POST "/dispatch/pool/$ticket_no/assign" "{\"masterId\":$MASTER_ID}" >/dev/null || return 1
  api POST "/tickets/$ticket_no/scan-bind" "{\"epc\":\"EPC-ACC-$suffix\"}" >/dev/null || return 1
  api POST "/tickets/$ticket_no/activate" >/dev/null || return 1
  local detail stage status
  detail=$(api GET "/orders/$order_no") || return 1
  stage=$(python3 -c "import json;d=json.loads('''$detail''');print(d['order']['stage'])")
  status=$(python3 -c "import json;d=json.loads('''$detail''');print(d['order']['status'])")
  [ "$stage" = "12" ] && [ "$status" = "DONE" ] || { FAIL_REASON="final $stage/$status"; return 1; }
  # 新开通订单的认证账号与套餐权威带宽
  local loid bw_exp
  loid=$(resolve_loid "$CUSTOMER_ID")
  [ -n "$loid" ] || { FAIL_REASON="loid not resolved for customer $CUSTOMER_ID"; return 1; }
  bw_exp=$(expected_bandwidth "$OFFER_ID") || return 1
  echo "  [dial] 订单=$order_no loid=$loid 期望带宽=$bw_exp" >&2
  # 三段拨号验证: 在服 ACCEPT(带宽=套餐) -> 停服 REJECT -> 复服 ACCEPT
  local out
  out=$(dial "$SECRET" "$loid") || { FAIL_REASON="phase1 no reply: $out"; return 1; }
  expect_dial "phase1-在服" "Access-Accept" "$bw_exp" "$out" >&2 || return 1
  api POST "/arrears/$CUSTOMER_ID/stop" >/dev/null || { FAIL_REASON="stop api"; return 1; }
  out=$(dial "$SECRET" "$loid") || { FAIL_REASON="phase2 no reply: $out"; return 1; }
  expect_dial "phase2-停服" "Access-Reject" "" "$out" >&2 || return 1
  api POST "/arrears/$CUSTOMER_ID/resume" >/dev/null || { FAIL_REASON="resume api"; return 1; }
  out=$(dial "$SECRET" "$loid") || { FAIL_REASON="phase3 no reply: $out"; return 1; }
  expect_dial "phase3-复服" "Access-Accept" "$bw_exp" "$out" >&2 || return 1
  echo "$order_no"
}

# run_negative: 错误共享密钥(RADIUS NAS「密码」)走同一在服断言,必判 FAIL。
# 返回 1=断言判死符合预期(负向自证成立); 2=错误密钥仍放行,认证失效; 3=无认证账号可测。
run_negative() {
  local loid out code
  loid=$(resolve_loid "$CUSTOMER_ID")
  if [ -z "$loid" ]; then
    echo "[dial-acceptance] negative FAIL: customer $CUSTOMER_ID 无认证账号" >&2
    return 3
  fi
  out=$(dial "${SECRET}-wrong" "$loid")
  code=$(dial_field "$out" code)
  if [ "$code" = "Access-Accept" ]; then
    echo "[dial-acceptance] DANGER: 错误密钥仍获 Access-Accept——响应认证器校验失效" >&2
    return 2
  fi
  echo "[dial-acceptance] negative: 错误密钥被拒(code=${code:-NO-REPLY})——同一断言此时必判 FAIL,脚本将以非 0 退出(自证)" >&2
  return 1
}

echo "拨号上网 E2E: $RUNS 轮 @ $RADIUS_ADDR (customer=$CUSTOMER_ID offer=$OFFER_ID)"
SECRET=$(read_secret) || { echo "FATAL: 无法从 $SSH_HOST 读 boss-aaa BOSS_AAA_SECRET(部署配置)" >&2; exit 2; }
echo "  [secret] 共享密钥已从部署配置读入内存(长度 ${#SECRET},不落盘不入仓库)"

if [ "$NEGATIVE" = "1" ]; then
  run_negative
  nrc=$?
  echo "negative 结果: 退出码 $nrc (1=断言判死符合预期 / 2=认证失效 / 3=无账号)"
  exit "$nrc"
fi

# 直营风控共存(同 mainchain-acceptance.sh):验收前临时关停 risk.direct.enabled,结束后恢复。
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

# 收尾自清理(与 mainchain 同一 acc_ 回收链路):造数不过夜。SKIP_CLEANUP=1 可跳过。
rc=0
if [ "${SKIP_CLEANUP:-0}" != "1" ]; then
  echo "收尾: 造数自清理"
  "$ROOT/scripts/ops/acceptance-cleanup.sh" --apply || rc=1
fi
echo "收尾: 孤儿巡检门禁"
"$ROOT/scripts/ops/db-patrol-gate.sh" || rc=1

# 恢复直营风控
if [ "$RISK_OFF" = "1" ]; then
  curl -sS -m 10 -X PUT "$API/params/risk.direct.enabled" -H "X-API-Key: $KEY" \
    -H "Content-Type: application/json" -d '{"value":"true"}' >/dev/null
  echo "  [risk] 直营风控已恢复"
fi

[ "$FAIL" -eq 0 ] && [ "$rc" -eq 0 ] && exit 0
exit 1