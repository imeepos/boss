#!/usr/bin/env bash
# verify-asset-scrap-confirm-e2e.sh -- 报废三要素确认端到端实测(P3-F/T3;102 真实部署,禁 mock)。
# 断言: F0 造数基线(资产1=有SN+绑标签,资产2=无SN未绑,SN 经 SQL 直置);
#       N1-N3 缺任一要素 -> 业务码 42200;W1-W3 错要素 -> 42200;
#       M2 报错信息指明要素且不回显服务端现值;X1 该空不空 -> 42200;
#       P1-P3 无SN/未绑资产空串确认报废成功(SCRAPPED,零 RECYCLE);
#       S0-S5 三要素正确报废成功+SCRAPPED+标签回收+RECYCLE 事件+SCRAPPED 轨迹+审计只落 SN 尾4位;
#       R1-R4 终态幂等:同请求重放成功且事件/轨迹/标签态零增量。
# 本库错误语义: HTTP 恒 200,422=业务码 42200(CodeInvalidParam),与 terms/httpx 对齐。
# 造数: acc_ 前缀族隔离(口径同 verify-asset-linkage-e2e);tag_events 无 FK 不受
#       acceptance-cleanup 覆盖,收尾自清+残留断言为零;幂等可重跑。
# 依赖: P3-E 迁移 000188(assets.sn)已部署;缺列时 F0 红灯提示依赖未就绪。
# 用法: scripts/e2e/verify-asset-scrap-confirm-e2e.sh [BASE_URL]  # 首参 http 开头=BASE_URL
# 环境: BASE_URL ADMIN_API_KEY SSH_HOST SKIP_CLEANUP SKIP_PATROL
# 依赖: curl python3 ssh(102 免密);鉴权 X-API-Key(test-accounts.json admin key)。
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

# sql: 102 断言/夹具 SQL(stdin 传 SQL 防叠引号,姿势同 mainchain-acceptance.sh)。
sql() { ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA"; }
cnt() { echo "$1" | sql | tr -d "[:space:]"; }

FAIL_REASON=""
# api METHOD PATH [JSON] -> stdout 业务 data;失败置 FAIL_REASON 返回 1(姿势同 linkage 脚本)。
api() {
  local method="$1" path="$2" data="" body out h1="X-API-Key: $KEY" h2="Content-Type: application/json"
  if [ $# -ge 3 ]; then data="$3"; fi
  if [ -n "$data" ]; then
    if ! body=$(curl -sS -m 30 -X "$method" "$API$path" -H "$h1" -H "$h2" -d "$data"); then FAIL_REASON="curl $path"; return 1; fi
  else
    if ! body=$(curl -sS -m 30 -X "$method" "$API$path" -H "$h1" -H "$h2"); then FAIL_REASON="curl $path"; return 1; fi
  fi
  out=$(python3 -c "import json,sys
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

PASS_N=0; FAIL_N=0; FAILED_IDS=""
ok() { PASS_N=$((PASS_N+1)); echo "PASS: [$1] $2"; }; bad() {
  FAIL_N=$((FAIL_N+1)); FAILED_IDS="$FAILED_IDS $1"
  echo "FAIL: [$1] $2" >&2
  echo "[e2e-scrap-confirm] ASSERTION FAILED id=$1 detail=$2" >&2; }
assert_eq() { if [ "$2" = "$3" ]; then ok "$1" "$2 ($4)"; else bad "$1" "期望=$2 实际=$3 上下文=$4"; fi; }

SUFFIX=""; BATCH_ID=0; TAG_ID=0; ASSET_ID=0; ASSET2_ID=0
SCRAP_CODE=""; SCRAP_REASON=""

# scrap_try ASSET_ID BODY:POST 报废并置 SCRAP_CODE(业务码)/SCRAP_REASON(422 reason)。
scrap_try() {
  local body
  SCRAP_CODE="-1"; SCRAP_REASON=""
  if ! body=$(curl -sS -m 30 -X POST "$API/assets/$1/scrap" -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d "$2"); then return; fi
  SCRAP_CODE=$(python3 -c "import json,sys
try: print(json.loads(sys.argv[1]).get('code',-2))
except Exception: print(-3)" "$body")
  SCRAP_REASON=$(python3 -c "import json,sys
try: d=json.loads(sys.argv[1]); r=d.get('data') or {}
except Exception: r={}
print(r.get('reason','') if isinstance(r,dict) else '')" "$body" 2>/dev/null)
}

# expect_422 ID LABEL BODY [MSG_HIT]:断言业务码 42200;MSG_HIT 非空时须出现在 reason。
expect_422() {
  scrap_try "$1" "$3"
  if [ "$SCRAP_CODE" != "42200" ]; then bad "$2" "期望=42200 实际=$SCRAP_CODE 请求=$3"; return; fi
  if [ $# -ge 4 ] && [ -n "$4" ] && echo "$SCRAP_REASON" | grep -q "$4"; then ok "$2" "42200 且 reason 命中 [$4]"; else ok "$2" "42200 (reason=$SCRAP_REASON)"; fi
}

boot_fixtures() { # 自举: 批次/标签/资产1(有SN绑标签)/资产2(无SN未绑),acc_ 前缀族
  local batch tag asset sn b1 b2
  batch=$(api POST /provision/asset-batches "{\"code\":\"RK-ACC-$SUFFIX\",\"name\":\"验收批次-$SUFFIX\",\"legalEntityId\":1}") || return 1
  BATCH_ID=$(j "$batch" id)
  EPC="30$(printf %s "$SUFFIX" | md5hex | tr -d " -" | cut -c1-22 | tr "a-f" "A-F")"
  tag=$(api POST /provision/tags "{\"tagNo\":\"T-ACC-$SUFFIX\",\"epcCode\":\"$EPC\",\"legalEntityId\":1,\"band\":\"UHF\",\"status\":\"UNBOUND\",\"battery\":\"100%\"}") || return 1
  TAG_ID=$(j "$tag" id)
  asset=$(api POST /provision/assets "{\"assetCode\":\"A-ACC-$SUFFIX\",\"batchId\":$BATCH_ID,\"legalEntityId\":1,\"legalEntityName\":\"验收主体\",\"type\":\"ONU\",\"status\":\"IN_STOCK\"}") || return 1
  ASSET_ID=$(j "$asset" id)
  asset=$(api POST /provision/assets "{\"assetCode\":\"A-ACC-$SUFFIX-B\",\"batchId\":$BATCH_ID,\"legalEntityId\":1,\"legalEntityName\":\"验收主体\",\"type\":\"ONU\",\"status\":\"IN_STOCK\"}") || return 1
  ASSET2_ID=$(j "$asset" id)
  if [ -z "$ASSET_ID" ] || [ "$ASSET_ID" = "None" ] || [ -z "$ASSET2_ID" ] || [ "$ASSET2_ID" = "None" ]; then
    FAIL_REASON="fixture ids incomplete asset1=$ASSET_ID asset2=$ASSET2_ID"; return 1; fi
  # SN 直置 SQL(不依赖 P3-E 创建 API 形态;缺列报错即依赖未部署,红灯留因)
  echo "UPDATE assets SET sn='SN-ACC-$SUFFIX' WHERE id=$ASSET_ID;" | sql || { FAIL_REASON="assets.sn 列缺失? P3-E 000188 未部署"; return 1; }
  echo "UPDATE tags SET bound_asset_id=$ASSET_ID, status='BOUND' WHERE id=$TAG_ID AND bound_asset_id IS NULL;" | sql || { FAIL_REASON="sql tag prebind"; return 1; }
  echo "UPDATE assets SET tag_id=$TAG_ID WHERE id=$ASSET_ID AND tag_id IS NULL;" | sql || { FAIL_REASON="sql asset prebind"; return 1; }
  sn=$(cnt "SELECT COALESCE(sn,'') FROM assets WHERE id=$ASSET_ID;")
  assert_eq "F0-sn" "SN-ACC-$SUFFIX" "$sn" "造数基线 assets.sn(id=$ASSET_ID)"
  b1=$(cnt "SELECT status || '|' || COALESCE(tag_id::text,'0') FROM assets WHERE id=$ASSET_ID;")
  assert_eq "F0-bind" "IN_STOCK|$TAG_ID" "$b1" "造数基线 资产1 绑标签(id=$TAG_ID)"
  b2=$(cnt "SELECT status || '|' || COALESCE(tag_id::text,'0') || '|' || COALESCE(sn,'<null>') FROM assets WHERE id=$ASSET2_ID;")
  assert_eq "F0-asset2" "IN_STOCK|0|<null>" "$b2" "造数基线 资产2 无SN未绑(id=$ASSET2_ID)"
}

negatives() { # 缺任一要素/错要素/该空不空 -> 全部 42200,且零副作用(资产仍 IN_STOCK)
  local CODE="A-ACC-$SUFFIX" SN="SN-ACC-$SUFFIX" TAG="T-ACC-$SUFFIX" st
  expect_422 "$ASSET_ID" "N1-缺编码" "{\"reason\":\"验收N1\",\"confirmSn\":\"$SN\",\"confirmTagNo\":\"$TAG\"}" "confirmAssetCode"
  expect_422 "$ASSET_ID" "N2-缺SN" "{\"reason\":\"验收N2\",\"confirmAssetCode\":\"$CODE\",\"confirmTagNo\":\"$TAG\"}" "confirmSn 必填"
  expect_422 "$ASSET_ID" "N3-缺标签号" "{\"reason\":\"验收N3\",\"confirmAssetCode\":\"$CODE\",\"confirmSn\":\"$SN\"}" "confirmTagNo 必填"
  expect_422 "$ASSET_ID" "W1-编码错" "{\"reason\":\"验收W1\",\"confirmAssetCode\":\"A-WRONG-$SUFFIX\",\"confirmSn\":\"$SN\",\"confirmTagNo\":\"$TAG\"}"
  expect_422 "$ASSET_ID" "W2-SN错" "{\"reason\":\"验收W2\",\"confirmAssetCode\":\"$CODE\",\"confirmSn\":\"SN-WRONG-$SUFFIX\",\"confirmTagNo\":\"$TAG\"}" "confirmSn"
  expect_422 "$ASSET_ID" "W3-标签号错" "{\"reason\":\"验收W3\",\"confirmAssetCode\":\"$CODE\",\"confirmSn\":\"$SN\",\"confirmTagNo\":\"T-WRONG-$SUFFIX\"}" "confirmTagNo"
  if echo "$SCRAP_REASON" | grep -q "SN-ACC-\|T-ACC-"; then
    bad "M2-回显泄露" "422 reason 泄露服务端现值: $SCRAP_REASON"
  else
    ok "M2-回显泄露" "reason 未回显正确 SN/标签号"
  fi
  st=$(cnt "SELECT status FROM assets WHERE id=$ASSET_ID;")
  assert_eq "N0-零副作用" "IN_STOCK" "$st" "全负例后资产1状态不变"
  expect_422 "$ASSET2_ID" "X1-该空不空" "{\"reason\":\"验收X1\",\"confirmAssetCode\":\"A-ACC-$SUFFIX-B\",\"confirmSn\":\"SN-X-$SUFFIX\",\"confirmTagNo\":\"\"}" "confirmSn 须为空串"
}

positive_bare() { # 资产2(无SN未绑) 空串确认报废成功,且无 RECYCLE
  local code st ev
  scrap_try "$ASSET2_ID" "{\"reason\":\"验收P1\",\"confirmAssetCode\":\"A-ACC-$SUFFIX-B\",\"confirmSn\":\"\",\"confirmTagNo\":\"\"}"
  assert_eq "P1-无SN空串报废" "0" "$SCRAP_CODE" "资产2 空串三要素确认"
  st=$(cnt "SELECT status FROM assets WHERE id=$ASSET2_ID;")
  assert_eq "P2-资产2终态" "SCRAPPED" "$st" "资产2 报废落终态"
  ev=$(cnt "SELECT count(*) FROM tag_events WHERE asset_id=$ASSET2_ID AND action='RECYCLE';")
  assert_eq "P3-零RECYCLE" "0" "$ev" "未绑标签报废不产生回收事件"
}

positive_tagged() { # 资产1 三要素正确报废成功+回收+事件+轨迹+审计脱敏
  local st tag ev lc a1 a2
  scrap_try "$ASSET_ID" "{\"reason\":\"验收S0\",\"confirmAssetCode\":\"A-ACC-$SUFFIX\",\"confirmSn\":\"SN-ACC-$SUFFIX\",\"confirmTagNo\":\"T-ACC-$SUFFIX\"}"
  assert_eq "S0-三要素报废" "0" "$SCRAP_CODE" "资产1 正确三要素"
  st=$(cnt "SELECT status FROM assets WHERE id=$ASSET_ID;")
  assert_eq "S1-SCRAPPED" "SCRAPPED" "$st" "资产1 报废落终态"
  tag=$(cnt "SELECT status || '|' || COALESCE(bound_asset_id::text,'<null>') FROM tags WHERE id=$TAG_ID;")
  assert_eq "S2-标签回收" "UNBOUND|<null>" "$tag" "标签强回收可复用"
  ev=$(cnt "SELECT count(*) FROM tag_events WHERE tag_id=$TAG_ID AND asset_id=$ASSET_ID AND action='RECYCLE';")
  assert_eq "S3-RECYCLE事件" "1" "$ev" "报废回收事件落库"
  lc=$(cnt "SELECT count(*) FROM asset_lifecycles WHERE asset_id=$ASSET_ID AND status='SCRAPPED';")
  assert_eq "S4-SCRAPPED轨迹" "1" "$lc" "资产生命周期 SCRAPPED 行"
  a1=$(cnt "SELECT count(*) FROM audit_logs WHERE target_type='asset' AND target_id='$ASSET_ID' AND detail::text LIKE '%confirmSnLast4%';")
  assert_eq "S5a-审计掩码" "1" "$a1" "审计附 confirmSnLast4"
  a2=$(cnt "SELECT count(*) FROM audit_logs WHERE target_type='asset' AND target_id='$ASSET_ID' AND detail::text LIKE '%SN-ACC-%';")
  assert_eq "S5b-审计不落全量SN" "0" "$a2" "审计不含完整 SN"
}

replay_idempotent() { # 终态幂等:同请求重放成功且零二次副作用
  local ev lc tag
  scrap_try "$ASSET_ID" "{\"reason\":\"验收S0\",\"confirmAssetCode\":\"A-ACC-$SUFFIX\",\"confirmSn\":\"SN-ACC-$SUFFIX\",\"confirmTagNo\":\"T-ACC-$SUFFIX\"}"
  assert_eq "R1-重放成功" "0" "$SCRAP_CODE" "已报废资产同请求重放"
  ev=$(cnt "SELECT count(*) FROM tag_events WHERE tag_id=$TAG_ID AND asset_id=$ASSET_ID AND action='RECYCLE';")
  assert_eq "R2-事件零增量" "1" "$ev" "重放不追加 RECYCLE"
  lc=$(cnt "SELECT count(*) FROM asset_lifecycles WHERE asset_id=$ASSET_ID AND status='SCRAPPED';")
  assert_eq "R3-轨迹零增量" "1" "$lc" "重放不追加 SCRAPPED 行"
  tag=$(cnt "SELECT status || '|' || COALESCE(bound_asset_id::text,'<null>') FROM tags WHERE id=$TAG_ID;")
  assert_eq "R4-标签态稳定" "UNBOUND|<null>" "$tag" "重放不回绑不翻状态"
}

cleanup_all() { # tag_events 自清(无 FK,cleanup 不覆盖) -> acceptance-cleanup -> 残留断言为零
  local q ev cls n rc
  echo "收尾: tag_events 自清(acc_ 资产系)"
  echo "DELETE FROM tag_events WHERE asset_id IN (SELECT id FROM assets WHERE asset_code LIKE 'A-ACC-%');" | sql \
    || bad "CLEAN-EV" "tag_events 自清失败(ssh/psql?)"
  if [ "$(env_or SKIP_CLEANUP 0)" != "1" ]; then
    echo "收尾: 造数自清理(acceptance-cleanup --apply,备份+单事务)"
    "$ROOT/scripts/ops/acceptance-cleanup.sh" --apply || bad "CLEANUP" "acceptance-cleanup --apply 失败"
  fi
  echo "收尾: 残留断言(应为零)"
  q=$(sql < "$ROOT/scripts/e2e/verify-asset-linkage-e2e-residue.sql")
  if [ -z "$q" ]; then bad "RES-db" "残留查询失败(ssh/psql 不可达),不能假装干净"; return; fi
  rc=0
  while IFS="|" read -r cls n; do
    [ -z "$cls" ] && continue
    if [ "$n" = "0" ]; then ok "RES-$cls" "残留=0"; else bad "RES-$cls" "残留 count=$n(查 acceptance-cleanup 口径)"; rc=1; fi
  done <<EOF
$q
EOF
  ev=$(cnt "SELECT count(*) FROM tag_events WHERE asset_id IN (SELECT id FROM assets WHERE asset_code LIKE 'A-ACC-%');")
  assert_eq "RES-tag-events" "0" "$ev" "报废事件自清残留"
  return "$rc"
}

echo "报废三要素确认端到端实测 @ $BASE_URL (suffix 将随造数生成)"
SUFFIX="$(date +%s)$RANDOM"
echo "== fixtures suffix=$SUFFIX =="
if boot_fixtures && negatives && positive_bare && positive_tagged && replay_idempotent; then
  echo "  断言完成"
else
  bad "RUN" "流程中断: $FAIL_REASON(造数 suffix=$SUFFIX 将由收尾清理回收)"
fi

rc=0
cleanup_all || rc=1
if [ "$(env_or SKIP_PATROL 0)" != "1" ]; then
  echo "收尾: 孤儿巡检门禁(db-patrol-gate)"
  "$ROOT/scripts/ops/db-patrol-gate.sh" || { bad "PATROL" "孤儿巡检超阈值"; rc=1; }
fi
echo "断言明细: PASS=$PASS_N FAIL=$FAIL_N 失败编号=[$FAILED_IDS ]"
if [ "$FAIL_N" -eq 0 ] && [ "$rc" -eq 0 ]; then
  echo "E2E-ASSET-SCRAP-CONFIRM RESULT: PASS pass=$PASS_N"
  exit 0
fi
echo "E2E-ASSET-SCRAP-CONFIRM RESULT: FAIL pass=$PASS_N fail=$FAIL_N failed=[$FAILED_IDS ]"
exit 1
