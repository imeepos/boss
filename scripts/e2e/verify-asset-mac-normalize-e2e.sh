#!/usr/bin/env bash
# verify-asset-mac-normalize-e2e.sh -- MAC 规范化存储+表达式唯一约束端到端实测(P4-T1,102 真实部署,禁 mock)。
# 迁移 000190/应用层 NormalizeMAC:写入归一为大写冒号规范形(AA:BB:CC:DD:EE:FF),输入仍接受
# 冒号/横杠/裸 hex 三形态。断言:
#   M1 创建 mac=aa-bb-cc-dd-ee-ff(小写横杠)成功
#   M2 直查库中值为 AA:BB:CC:DD:EE:FF(归一落库)
#   M3 再创建 mac=AA:BB:CC:DD:EE:FF 被拒 40900(冒号同值冲突)
#   M4 再创建 mac=AABBCCDDEEFF 裸 hex 全同值也被拒 40900(表达式索引归一空间一致)
#   M5 非法格式 AA:BB:CC 仍 42200(格式闸门沿用,触库前短路)
#   M6 组形横杠 aabb-ccdd-eeff 拒 42200(非六组两位形态,格式校验沿用;验收所谓
#      小写横杠输入的规范展开即 M1 的 aa-bb-cc-dd-ee-ff 六组形,契约 fields.md 口径)
#   M7 编辑 PUT mac=aabbccddeef0 裸 hex 生效,库中归一为 AA:BB:CC:DD:EE:F0
#   RES 收尾清理造数后残留断言为零(脚本可重复执行,幂等靠前缀清理)
# 信号: 每条断言独立输出 "PASS: [编号] ..." / "FAIL: [编号] ...";收尾 "E2E-ASSET-MAC-NORMALIZE RESULT: ..." 可 grep。
# 用法: scripts/e2e/verify-asset-mac-normalize-e2e.sh [BASE_URL]
# 环境: BASE_URL ADMIN_API_KEY SSH_HOST SKIP_CLEANUP=1
# 依赖: curl python3 ssh(102 免密);鉴权 X-API-Key(test-accounts.json admin key)。
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
PREFIX="ACC-MAC"

# sql: 102 断言/清理 SQL(stdin 传 SQL 防叠引号,姿势同 verify-asset-identity-epc-e2e.sh)。
sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }
sqlval() { sql < <(echo "$1") | tr -d '[:space:]'; }

FAIL_REASON=""
# api METHOD PATH BODY -> 业务 data JSON;业务码非 0/200 输出空(失败留 body 供断言)。
api() {
  local method="$1" path="$2" body="$3" out
  out=$(curl -sS -m 30 -X "$method" "$API$path" -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d "$body") || { FAIL_REASON="curl $path"; return 1; }
  LAST_BODY="$out"
  out=$(python3 - "$out" <<PY
import json,sys
d=json.loads(sys.argv[1])
print(json.dumps(d.get("data")) if d.get("code") in (0,200) else "")
PY
)
  if [ -z "$out" ]; then FAIL_REASON="api $2 -> $(echo "$LAST_BODY" | head -c 200)"; return 1; fi
  echo "$out"
}

# jcode/jdata: 原始响应体顶层 code / data 内字段(409/422 断言走原始体,不经 api() 短路)。
jcode() { python3 - "$1" <<PY
import json,sys
print(json.loads(sys.argv[1]).get("code",""))
PY
}
jdata() { python3 - "$1" "$2" <<PY
import json,sys
d=json.loads(sys.argv[1]).get("data")
print(d.get(sys.argv[2],"") if isinstance(d,dict) else "")
PY
}

PASS_N=0; FAIL_N=0; FAILED_IDS=""
ok() { PASS_N=$((PASS_N+1)); echo "PASS: [$1] $2"; }
bad() { FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"; echo "FAIL: [$1] $2" >&2; }
assert_eq() { if [ "$2" = "$3" ]; then ok "$1" "$2 ($4)"; else bad "$1" "期望=$2 实际=$3 上下文=$4"; fi }

# expect_reject ID BODY WANT_CODE WANT_SUBSTR: 期望业务码与 data.reason 子串(冲突字段名)。
expect_reject() {
  local id="$1" body="$2" want="$3" substr="$4" code reason
  code=$(jcode "$body")
  if [ "$code" != "$want" ]; then bad "$id" "期望 code=$want 实际=$code body=$(echo "$body" | head -c 160)"; return; fi
  ok "$id" "拒绝语义 code=$want"
  if [ -n "$substr" ]; then
    reason=$(jdata "$body" reason)
    case "$reason" in
      *"$substr"*) ok "$id-f" "reason 含冲突字段名 <$substr>" ;;
      *) bad "$id-f" "reason 缺 <$substr>: $reason" ;;
    esac
  fi
}

sweep_sql() { # 前缀族清理(先子后主,幂等)
  sql <<SQL
DELETE FROM asset_lifecycles WHERE asset_id IN (SELECT id FROM assets WHERE asset_code LIKE '$PREFIX-%');
DELETE FROM assets WHERE asset_code LIKE '$PREFIX-%';
DELETE FROM asset_batches WHERE code LIKE 'RK-$PREFIX-%';
SQL
}

BATCH_ID=""; AID1=""; LAST_BODY=""
pre_clean() { echo "前置清理: $PREFIX 残留"; sweep_sql; }
boot_fixtures() {
  local batch
  SUFFIX="$(date +%s)$RANDOM"
  batch=$(api POST /asset-batches "$(printf '{"code":"RK-%s-%s","name":"MAC 归一E2E批次","legalEntityId":1}' "$PREFIX" "$SUFFIX")") || return 1
  BATCH_ID=$(printf '%s' "$batch" | python3 -c "import json,sys;print(json.load(sys.stdin).get('id',''))")
  [ -n "$BATCH_ID" ] || { FAIL_REASON="batch id empty"; return 1; }
}

t_mac() {
  # M1 创建小写横杠形成功(归一入口)
  local body1
  body1="$(printf '{"assetCode":"%s","batchId":%s,"type":"ONU","mac":"aa-bb-cc-dd-ee-ff"}' "$PREFIX-AS-1-$SUFFIX" "$BATCH_ID")"
  local out
  out=$(api POST /assets "$body1") || { bad "M1" "建档失败: $FAIL_REASON"; return 1; }
  AID1=$(printf '%s' "$out" | python3 -c "import json,sys;print(json.load(sys.stdin).get('id',''))")
  assert_eq "M1" "ok" "$([ -n "$AID1" ] && echo ok)" "小写横杠 mac 建档成功 id=$AID1"

  # M2 直查库中值 = 大写冒号规范形
  local stored
  stored=$(sqlval "SELECT COALESCE(mac,'') FROM assets WHERE id=$AID1;")
  assert_eq "M2" "AA:BB:CC:DD:EE:FF" "$stored" "归一落库"

  # M3 冒号形同值再建档 → 40900 + reason 含 mac
  local body3 body4 body5 body6
  body3="$(printf '{"assetCode":"%s","batchId":%s,"type":"ONU","mac":"AA:BB:CC:DD:EE:FF"}' "$PREFIX-AS-DUP-$SUFFIX" "$BATCH_ID")"
  LAST_BODY=$(curl -sS -m 30 -X POST "$API/assets" -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d "$body3") || { bad "M3" "curl"; return 1; }
  expect_reject "M3" "$LAST_BODY" "40900" "mac"

  # M4 裸 hex 全同值再建档 → 40900(表达式唯一索引与应用层同一归一空间)
  body4="$(printf '{"assetCode":"%s","batchId":%s,"type":"ONU","mac":"AABBCCDDEEFF"}' "$PREFIX-AS-BARE-$SUFFIX" "$BATCH_ID")"
  LAST_BODY=$(curl -sS -m 30 -X POST "$API/assets" -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d "$body4") || { bad "M4" "curl"; return 1; }
  expect_reject "M4" "$LAST_BODY" "40900" "mac"

  # M5 非法格式仍 42200(格式闸门沿用,触库前短路)
  body5="$(printf '{"assetCode":"%s","batchId":%s,"type":"ONU","mac":"AA:BB:CC"}' "$PREFIX-AS-BAD-$SUFFIX" "$BATCH_ID")"
  LAST_BODY=$(curl -sS -m 30 -X POST "$API/assets" -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d "$body5") || { bad "M5" "curl"; return 1; }
  expect_reject "M5" "$LAST_BODY" "42200" ""

  # M6 组形横杠(非六组两位)仍拒:格式校验沿用,归一只改写不放宽
  body6="$(printf '{"assetCode":"%s","batchId":%s,"type":"ONU","mac":"aabb-ccdd-eeff"}' "$PREFIX-AS-GRP-$SUFFIX" "$BATCH_ID")"
  LAST_BODY=$(curl -sS -m 30 -X POST "$API/assets" -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d "$body6") || { bad "M6" "curl"; return 1; }
  expect_reject "M6" "$LAST_BODY" "42200" ""

  # M7 编辑路径:裸 hex 归一生效(PUT 指针语义,其他键缺省保持)
  local body7 stored7
  body7="$(printf '{"mac":"aabbccddeef0"}')"
  out=$(api PUT "/assets/$AID1" "$body7") || { bad "M7" "编辑失败: $FAIL_REASON"; return 1; }
  stored7=$(sqlval "SELECT COALESCE(mac,'') FROM assets WHERE id=$AID1;")
  assert_eq "M7" "AA:BB:CC:DD:EE:F0" "$stored7" "编辑裸 hex 归一落库"
}

cleanup() {
  sweep_sql || { bad "CLEAN" "清理 SQL 失败"; return 1; }
  local q rc=0
  q=$(sql <<SQL
SELECT 'assets', count(*) FROM assets WHERE asset_code LIKE '$PREFIX-%'
UNION ALL SELECT 'batches', count(*) FROM asset_batches WHERE code LIKE 'RK-$PREFIX-%';
SQL
)
  while IFS="|" read -r cls n; do
    [ -z "$cls" ] && continue
    if [ "$n" = "0" ]; then ok "RES-$cls" "残留=0"; else bad "RES-$cls" "残留 count=$n"; rc=1; fi
  done <<RES
$q
RES
  return $rc
}

echo "MAC 规范化存储+表达式唯一约束 e2e @ $BASE_URL"
pre_clean
if boot_fixtures && t_mac; then
  echo "  断言完成"
else
  bad "RUN" "流程中断: $FAIL_REASON"
fi
rc=0
if [ "$(env_or SKIP_CLEANUP 0)" != "1" ]; then
  cleanup || rc=1
fi
echo "断言明细: PASS=$PASS_N FAIL=$FAIL_N 失败编号=[$FAILED_IDS ]"
if [ "$FAIL_N" -eq 0 ] && [ "$rc" -eq 0 ]; then
  echo "E2E-ASSET-MAC-NORMALIZE RESULT: PASS pass=$PASS_N"
  exit 0
fi
echo "E2E-ASSET-MAC-NORMALIZE RESULT: FAIL pass=$PASS_N fail=$FAIL_N failed=[$FAILED_IDS ]"
exit 1