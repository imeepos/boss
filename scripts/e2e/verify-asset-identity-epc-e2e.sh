#!/usr/bin/env bash
# e2e-asset-identity-epc.sh -- 资产身份三要素 + EPC 校验端到端实测(102 真实部署,禁 mock)。
# P3-T2/迁移 000188:建档接收 SN/MAC/LOID;同 SN 再建 40900(reason 含冲突字段名);
# 非法 MAC 42200;非法 EPC(位数不足/头部字节不符)42200;合法 24-hex EPC 入库统一大写。
# 造数: ACC-ID- 前缀族隔离,前缀清理 + 残留断言为零,幂等可重跑。
# 信号: 每条断言独立输出 PASS/FAIL,收尾 "E2E-ASSET-IDENTITY-EPC RESULT: ..." 可 grep。
# 用法: scripts/e2e/verify-asset-identity-epc-e2e.sh [BASE_URL]
# 环境: BASE_URL ADMIN_API_KEY SSH_HOST   依赖: curl python3 ssh(102 免密)
set -u

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
source "$ROOT/scripts/ops/acceptance-lock.sh"
acquire_acceptance_lock || exit 1
trap release_acceptance_lock EXIT

env_or() { local v; v=$(printenv "$1" 2>/dev/null); if [ -n "$v" ]; then echo "$v"; else echo "$2"; fi; }
# md5hex: macOS md5 / Linux md5sum 双通(102 宿主 cron 本机执行依赖,行为不变)。
md5hex() { if command -v md5 > /dev/null 2>&1; then md5; else md5sum | cut -d" " -f1; fi; }
ARG1=""; if [ $# -ge 1 ]; then ARG1="$1"; fi
BASE_URL=$(env_or BASE_URL "http://192.168.0.102:28080")
case "$ARG1" in http*) BASE_URL="$ARG1" ;; esac
API="$BASE_URL/api/admin/v1"
KEY=$(env_or ADMIN_API_KEY "$(python3 -c "import json;print(json.load(open('$ROOT/.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key'])")")
SSH_HOST=$(env_or SSH_HOST "imeepos@192.168.0.102")
PREFIX="ACC-ID"

# sql: 102 断言/清理 SQL(stdin 传 SQL 防叠引号,姿势同 verify-asset-linkage-e2e.sh)。
sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }

FAIL_REASON=""
# api METHOD PATH [JSON] -> 业务 data JSON(业务码非 0/200 视为失败)。
api() {
  local method="$1" path="$2" data="" body out
  if [ $# -ge 3 ]; then data="$3"; fi
  if [ -n "$data" ]; then
    body=$(curl -sS -m 30 -X "$method" "$API$path" -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d "$data") || { FAIL_REASON="curl $path"; return 1; }
  else
    body=$(curl -sS -m 30 "$API$path" -H "X-API-Key: $KEY") || { FAIL_REASON="curl $path"; return 1; }
  fi
  out=$(python3 -c "import json,sys;d=json.loads(sys.argv[1]);print(json.dumps(d.get('data')) if d.get('code') in (0,200) else '')" "$body")
  if [ -z "$out" ]; then FAIL_REASON="api $2 -> $(echo "$body" | head -c 200)"; echo "[api-fail] $FAIL_REASON" >&2; return 1; fi
  echo "$out"
}
j() { python3 -c "import json,sys;print(json.loads(sys.argv[1]).get(sys.argv[2],''))" "$1" "$2"; }

PASS_N=0; FAIL_N=0; FAILED_IDS=""
ok() { PASS_N=$((PASS_N+1)); echo "PASS: [$1] $2"; }
bad() { FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"; echo "FAIL: [$1] $2" >&2; }
assert_eq() { if [ "$2" = "$3" ]; then ok "$1" "$2 ($4)"; else bad "$1" "期望=$2 实际=$3 上下文=$4"; fi }
# expect_reject ID PATH JSON WANT_CODE WANT_SUBSTR: 期望业务码与 reason 子串。
expect_reject() {
  local id="$1" path="$2" data="$3" want="$4" substr="$5" body code reason
  body=$(curl -sS -m 30 -X POST "$API$path" -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d "$data") || { bad "$id" "curl 失败"; return; }
  code=$(python3 -c "import json,sys;print(json.loads(sys.argv[1]).get('code',''))" "$body")
  reason=$(python3 -c "import json,sys;d=json.loads(sys.argv[1]);r=d.get('data',{}).get('reason','') if isinstance(d.get('data'),dict) else '';print(r)" "$body")
  if [ "$code" = "$want" ]; then ok "$id" "拒绝语义 code=$want ($path)"; else bad "$id" "期望 code=$want 实际=$code body=$(echo "$body" | head -c 160)"; return; fi
  if [ -n "$substr" ]; then
    case "$reason" in
      *"$substr"*) ok "$id-f" "reason 含冲突字段名 <$substr>" ;;
      *) bad "$id-f" "reason 缺 <$substr>: $reason" ;;
    esac
  fi
}

sweep_sql() { # 前缀族清理(先子后主)
  sql <<SQL
DELETE FROM tag_events WHERE tag_id IN (SELECT id FROM tags WHERE tag_no LIKE '$PREFIX-%');
DELETE FROM asset_lifecycles WHERE asset_id IN (SELECT id FROM assets WHERE asset_code LIKE '$PREFIX-%');
DELETE FROM assets WHERE asset_code LIKE '$PREFIX-%';
DELETE FROM tags WHERE tag_no LIKE '$PREFIX-%';
DELETE FROM asset_batches WHERE code LIKE 'RK-$PREFIX-%';
SQL
}

BATCH_ID=""; AID1=""; AID4=""; TAGID3=""
pre_clean() { echo "前置清理: $PREFIX 残留"; sweep_sql; }
boot_fixtures() {
  local batch
  SUFFIX="$(date +%s)$RANDOM"
  batch=$(api POST /asset-batches "{\"code\":\"RK-$PREFIX-$SUFFIX\",\"name\":\"身份列E2E批次\",\"legalEntityId\":1}") || return 1
  BATCH_ID=$(j "$batch" id)
  [ -n "$BATCH_ID" ] || { FAIL_REASON="batch id empty"; return 1; }
}

t_identity() { # I1-I4
  local out sn h mac1 row1
  sn="$PREFIX-SN-$SUFFIX"
  h=$(printf %s "$SUFFIX" | md5hex | tr -d " -" | cut -c1-8 | tr "a-f" "A-F")
  mac1="AC:AC:${h:0:2}:${h:2:2}:${h:4:2}:${h:6:2}"
  out=$(api POST /assets "{\"assetCode\":\"$PREFIX-AS-$SUFFIX\",\"batchId\":$BATCH_ID,\"type\":\"ONU\",\"sn\":\"  $sn  \",\"mac\":\"$mac1\",\"loid\":\"$PREFIX-LOID-$SUFFIX\"}") || return 1
  AID1=$(j "$out" id)
  assert_eq "I1" "ok" "$([ -n "$AID1" ] && echo ok)" "建档带 SN/MAC/LOID 成功 id=$AID1"
  row1=$(sql <<SQL
SELECT COALESCE(sn,'') || '|' || COALESCE(mac,'') || '|' || COALESCE(loid,'') FROM assets WHERE id=$AID1;
SQL
)
  assert_eq "I1b" "$sn|$mac1|$PREFIX-LOID-$SUFFIX" "$row1" "落库: SN 去空格, MAC 原样入库"
  expect_reject "I2" /assets "{\"assetCode\":\"$PREFIX-AS-DUP-$SUFFIX\",\"batchId\":$BATCH_ID,\"type\":\"ONU\",\"sn\":\"$sn\"}" "40900" "sn"
  expect_reject "I3" /assets "{\"assetCode\":\"$PREFIX-AS-BADMAC-$SUFFIX\",\"batchId\":$BATCH_ID,\"type\":\"ONU\",\"mac\":\"AA:BB:CC\"}" "42200" ""
  out=$(api POST /assets "{\"assetCode\":\"$PREFIX-AS-DASH-$SUFFIX\",\"batchId\":$BATCH_ID,\"type\":\"ONU\",\"mac\":\"AB-AB-01-23-45-67\"}") || return 1
  AID4=$(j "$out" id)
  assert_eq "I4" "ok" "$([ -n "$AID4" ] && echo ok)" "横杠分隔 MAC 亦合法 id=$AID4"
}

t_epc() { # E1-E3
  local hex out stored want_up
  expect_reject "E1" /tags "{\"tagNo\":\"$PREFIX-T1-$SUFFIX\",\"epcCode\":\"30ABC\",\"legalEntityId\":1,\"band\":\"UHF\"}" "42200" ""
  hex="31$(printf %s "$SUFFIX" | md5hex | tr -d " -" | cut -c1-22 | tr "a-f" "A-F")"
  expect_reject "E2" /tags "{\"tagNo\":\"$PREFIX-T2-$SUFFIX\",\"epcCode\":\"$hex\",\"legalEntityId\":1,\"band\":\"UHF\"}" "42200" ""
  hex="30$(printf %s "$SUFFIX-x" | md5hex | tr -d " -" | cut -c1-22)"
  want_up=$(echo "$hex" | tr "a-f" "A-F")
  out=$(api POST /tags "{\"tagNo\":\"$PREFIX-T3-$SUFFIX\",\"epcCode\":\"$hex\",\"legalEntityId\":1,\"band\":\"UHF\"}") || return 1
  TAGID3=$(j "$out" id)
  assert_eq "E3" "ok" "$([ -n "$TAGID3" ] && echo ok)" "合法小写 24-hex EPC 建标签成功 id=$TAGID3"
  stored=$(sql <<SQL
SELECT epc_code FROM tags WHERE id=$TAGID3;
SQL
)
  assert_eq "E3b" "$want_up" "$stored" "入库统一大写"
}

cleanup() {
  sweep_sql || { bad "CLEAN" "清理 SQL 失败"; return 1; }
  local q rc=0
  q=$(sql <<SQL
SELECT 'assets', count(*) FROM assets WHERE asset_code LIKE '$PREFIX-%'
UNION ALL SELECT 'tags', count(*) FROM tags WHERE tag_no LIKE '$PREFIX-%'
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

echo "资产身份三要素 + EPC 校验 e2e @ $BASE_URL"
pre_clean
if boot_fixtures && t_identity && t_epc; then
  echo "  断言完成"
else
  bad "RUN" "流程中断: $FAIL_REASON"
fi
rc=0
cleanup || rc=1
echo "断言明细: PASS=$PASS_N FAIL=$FAIL_N 失败编号=[$FAILED_IDS ]"
if [ "$FAIL_N" -eq 0 ] && [ "$rc" -eq 0 ]; then
  echo "E2E-ASSET-IDENTITY-EPC RESULT: PASS pass=$PASS_N"
  exit 0
fi
echo "E2E-ASSET-IDENTITY-EPC RESULT: FAIL pass=$PASS_N fail=$FAIL_N failed=[$FAILED_IDS ]"
exit 1
