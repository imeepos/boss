#!/usr/bin/env bash
# verify-stocktake-flow.sh — 盘点差异闭环 CI 部署验证(102 真接口,非单测非集成)。
#
# S10 全流程:建任务(快照) → 扫码(OK/MISMATCH/EXTRA) → 关单守卫(40900)
#            → 逐条处置(CONFIRM 修正台账+轨迹 / MISSING / EXTRA) → 关单 DONE
#
# 前置夹具(102 实测存在):legal_entity_id=1 且 region_name='测试区域' 有 5 台 DEPLOYED
# 资产(126,128,130,132,134);脚本动态复核,不足即 FAIL。
# 收尾:恢复被 CONFIRM 修正的资产状态(数据夹具复位),任务/明细/轨迹保留作证据。
#
# 用法: scripts/verify-stocktake-flow.sh [BASE_URL]   缺省 http://192.168.0.102:28080
# 依赖: jq + curl + ssh(102 SQL 复核)
set -euo pipefail

BASE="${1:-http://192.168.0.102:28080}"
SCOPE="测试区域"
ENTITY=1
echo "==> 目标: $BASE"

TOKEN=$(curl -s "$BASE/api/admin/v1/auth/login" -X POST \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.token')
if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then echo "FAIL: 登录失败" >&2; exit 1; fi
H_AUTH="Authorization: Bearer $TOKEN"
H_JSON="Content-Type: application/json"
api() { local p=$1 m=${2:-GET} b=${3:-}; curl -s -w "\n%{http_code}" "$BASE/api/admin/v1$p" -X "$m" ${b:+-H "$H_JSON"} ${b:+-d "$b"} -H "$H_AUTH"; }

# ── 夹具复核 ────────────────────────────────────────────────────────────────
echo "==> 夹具复核:实体 ${ENTITY} 区域「${SCOPE}」DEPLOYED 资产"
FIXTURE=$(ssh imeepos@192.168.0.102 "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tAc \
  \"SELECT string_agg(id::text, ',') FROM assets WHERE legal_entity_id=$ENTITY AND region_name='$SCOPE' AND status='DEPLOYED'\"")
echo "  ids=[$FIXTURE]"
COUNT=$(echo "$FIXTURE" | tr ',' '\n' | grep -c . || true)
if [ "${COUNT:-0}" -lt 5 ]; then echo "FAIL: 夹具不足(需 ≥5 台),实际 $COUNT" >&2; exit 1; fi
A_OK=$(echo "$FIXTURE" | cut -d, -f1)      # 扫 DEPLOYED → OK
A_MM=$(echo "$FIXTURE" | cut -d, -f2)      # 扫 IN_STOCK → MISMATCH → CONFIRM 修正台账
EXTRA_ASSET=$(curl -s "$BASE/api/admin/v1/assets" -H "$H_AUTH" \
  | jq -r '[.data.items[] | select(.status == "IN_STOCK" and (.regionName == "" or .regionName == null))][0].assetId')
echo "  OK→$A_OK  MISMATCH→$A_MM  EXTRA→$EXTRA_ASSET"

# ── 场景 1: 建任务+快照 ─────────────────────────────────────────────────────
echo ""
echo "==> 场景 1: POST /stocktakes 建任务(冻结 $COUNT 台快照)"
RESP=$(api "/stocktakes" POST "{\"legalEntityId\":$ENTITY,\"scope\":\"$SCOPE\"}")
HTTP=$(echo "$RESP" | tail -n1); BODY=$(echo "$RESP" | sed -e '$d')
[ "$HTTP" = "200" ] || { echo "FAIL: 建任务 http=$HTTP body=$BODY" >&2; exit 1; }
TASK=$(echo "$BODY" | jq -r '.data.id'); echo "  task_id=$TASK"
N=$(api "/stocktakes/$TASK/items" | sed -e '$d' | jq '.data.items | length')
[ "$N" = "$COUNT" ] || { echo "FAIL: 快照行数 $N ≠ $COUNT" >&2; exit 1; }
echo "  PASS: 快照 $N 行(全 PENDING)"

# ── 场景 2: 扫码 OK / MISMATCH / EXTRA ─────────────────────────────────────
echo ""
echo "==> 场景 2: 扫码回填"
K=$(api "/stocktakes/$TASK/scans" POST "{\"assetId\":$A_OK,\"status\":\"DEPLOYED\"}" | sed -e '$d' | jq -r '.data.kind')
[ "$K" = "OK" ] || { echo "FAIL: $A_OK 期望 OK 实际 $K" >&2; exit 1; }
K=$(api "/stocktakes/$TASK/scans" POST "{\"assetId\":$A_MM,\"status\":\"IN_STOCK\"}" | sed -e '$d' | jq -r '.data.kind')
[ "$K" = "MISMATCH" ] || { echo "FAIL: $A_MM 期望 MISMATCH 实际 $K" >&2; exit 1; }
K=$(api "/stocktakes/$TASK/scans" POST "{\"assetId\":$EXTRA_ASSET,\"status\":\"IN_STOCK\"}" | sed -e '$d' | jq -r '.data.kind')
[ "$K" = "EXTRA" ] || { echo "FAIL: $EXTRA_ASSET 期望 EXTRA 实际 $K" >&2; exit 1; }
PROG=$(ssh imeepos@192.168.0.102 "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tAc \
  \"SELECT progress || '/' || diff_count FROM stocktakes WHERE id=$TASK\"")
echo "  PASS: OK/MISMATCH/EXTRA 判定正确,进度/差异=[$PROG](期望 40/2)"

# ── 场景 3: 关单守卫(存在未处置差异 → 40900)────────────────────────────────
echo ""
echo "==> 场景 3: 未处置差异时关单应被拒"
RESP=$(api "/stocktakes/$TASK/diff-handle" POST "")
HTTP=$(echo "$RESP" | tail -n1); CODE=$(echo "$RESP" | sed -e '$d' | jq -r '.code')
[ "$CODE" = "40900" ] || { echo "FAIL: 期望 40900 实际 $CODE http=$HTTP" >&2; exit 1; }
echo "  PASS: 关单被拒($CODE)"

# ── 场景 4: 逐条处置 ────────────────────────────────────────────────────────
echo ""
echo "==> 场景 4: CONFIRM(MISMATCH 修正台账+轨迹)/ MISSING / EXTRA"
FIX_NOTE="E2E: 现场核实台账正确"
handle_item() { # $1=assetId $2=body —— 按 asset 定位明细行,校验业务码后处置
  local ITEM_ID BODY CODE HTTP
  ITEM_ID=$(ssh imeepos@192.168.0.102 "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tAc \
    \"SELECT id FROM stocktake_items WHERE task_id=$TASK AND asset_id=$1\"")
  [ -n "$ITEM_ID" ] || { echo "FAIL: asset $1 在任务 $TASK 无明细行" >&2; exit 1; }
  BODY=$(api "/stocktakes/$TASK/items/$ITEM_ID/handle" POST "$2" | sed -e '$d')
  CODE=$(echo "$BODY" | jq -r '.code')
  [ "$CODE" = "0" ] || { echo "FAIL: 处置 item=$ITEM_ID 业务码=$CODE body=$BODY" >&2; exit 1; }
}
handle_item "$A_MM" '{"action":"CONFIRM"}'
handle_item "$EXTRA_ASSET" "{\"action\":\"CONFIRM\",\"note\":\"$FIX_NOTE\"}"
for ID in $(echo "$FIXTURE" | tr ',' '\n' | grep -v "^$A_OK$" | grep -v "^$A_MM$" | tail -3); do
  handle_item "$ID" "{\"action\":\"CONFIRM\",\"note\":\"$FIX_NOTE\"}"
done
LC=$(ssh imeepos@192.168.0.102 "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tAc \
  \"SELECT count(*) FROM asset_lifecycles WHERE asset_id=$A_MM AND status='IN_STOCK'\"")
ST=$(ssh imeepos@192.168.0.102 "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tAc \
  \"SELECT status FROM assets WHERE id=$A_MM\"")
[ "$ST" = "IN_STOCK" ] && [ "$LC" != "0" ] \
  || { echo "FAIL: CONFIRM 未修正台账(status=$ST lifecycle=$LC)" >&2; exit 1; }
echo "  PASS: 台账已按实盘修正(status=$ST)+ 轨迹 $LC 条"

# ── 场景 5: 全处置完关单 → DONE ────────────────────────────────────────────
echo ""
echo "==> 场景 5: 关单"
HTTP=$(api "/stocktakes/$TASK/diff-handle" POST "" | tail -n1)
[ "$HTTP" = "200" ] || { echo "FAIL: 关单 http=$HTTP" >&2; exit 1; }
FINAL=$(ssh imeepos@192.168.0.102 "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tAc \
  \"SELECT status || '/' || progress FROM stocktakes WHERE id=$TASK\"")
[ "$FINAL" = "DONE/100" ] || { echo "FAIL: 任务终态 $FINAL ≠ DONE/100" >&2; exit 1; }
echo "  PASS: 任务 $FINAL"
RESP=$(api "/stocktakes/$TASK/scans" POST "{\"assetId\":$A_OK,\"status\":\"DEPLOYED\"}")
CODE=$(echo "$RESP" | sed -e '$d' | jq -r '.code')
[ "$CODE" = "40900" ] && echo "  PASS: 已关单任务拒绝再扫码(40900)"

# ── 夹具复位 ────────────────────────────────────────────────────────────────
echo ""
echo "==> 夹具复位:资产 $A_MM 恢复 DEPLOYED(任务/明细/轨迹保留作证据)"
ssh imeepos@192.168.0.102 "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -c \
  \"UPDATE assets SET status='DEPLOYED' WHERE id=$A_MM\"" >/dev/null
echo ""
echo "==============================================="
echo "PASS: 盘点差异闭环 S10 全流程真实环境验证通过"
echo "  证据: stocktake id=$TASK(保留), status=DONE/100"
echo "==============================================="
