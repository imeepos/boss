#!/usr/bin/env bash
# 用户实名认证流程 真环境冒烟(默认打 102 部署 http://192.168.0.102:28080):
#   注册建档 → 后台代录 PENDING → 待办创建 → 全部已读 → FAIL 驳回(回执+待办办结)
#   → 重提 PENDING → 同 id 待办复活且未读数回升(P0 复活语义实证)→ PASS 通过
#   → customers.real_name_status=VERIFIED;全程造数收尾自清理(不过夜红线)。
# 用法: bash scripts/realname-e2e-smoke.sh   (可覆盖 BOSS_BASE/BOSS_SSH/PG_CONTAINER)
set -uo pipefail

BASE="${BOSS_BASE:-http://192.168.0.102:28080}"
SSH_TARGET="${BOSS_SSH:-imeepos@192.168.0.102}"
PG_CONTAINER="${PG_CONTAINER:-boss-infra-postgres-1}"
PASS_CNT=0; FAIL_CNT=0; LOG="/tmp/realname-e2e-$(date +%s).log"

log() { echo "[$(date +%H:%M:%S)] $*" | tee -a "$LOG"; }
say() { echo "     $*" | tee -a "$LOG"; }

# sqlexec 经 stdin 送 SQL 进库容器,避免 ssh 双层引号叠炸(教训 9a)。
sqlexec() {
  printf '%s' "$1" | ssh "$SSH_TARGET" "docker exec -i $PG_CONTAINER psql -U boss -d boss -tA" 2>>"$LOG"
}

api() { # api METHOD PATH [JSON]
  local m="$1" p="$2" body="${3:-}"
  if [ -n "$body" ]; then
    curl -sS -m 20 "$BASE$p" -X "$m" -H 'Content-Type: application/json' \
      ${TOK:+-H "Authorization: Bearer $TOK"} -d "$body"
  else
    curl -sS -m 20 "$BASE$p" -X "$m" ${TOK:+-H "Authorization: Bearer $TOK"}
  fi
}

expect() { # expect LABEL ACTUAL EXPECTED
  local label="$1" actual="$2" want="$3"
  if [ "$actual" = "$want" ]; then
    PASS_CNT=$((PASS_CNT+1)); log "OK   $label = $actual"
  else
    FAIL_CNT=$((FAIL_CNT+1)); log "FAIL $label: got [$actual] want [$want]"
  fi
}

cleanup() { # 带上自造数据主键逐一删除;FK 子表在前
  [ -n "${CID:-}" ] || return 0
  sqlexec "DELETE FROM verifications WHERE subject_type='customer' AND subject_id=$CID;
DELETE FROM portal_messages WHERE customer_id=$CID;
DELETE FROM admin_notifications WHERE ref_type='realname' AND ref_id='customer/$CID';
DELETE FROM audit_logs WHERE target_id IN ('$CID','$RID') AND action LIKE '%realname%';
DELETE FROM audit_logs WHERE action LIKE '%registration%' AND target_id='$RID';
DELETE FROM customers WHERE id=$CID;
DELETE FROM customer_registrations WHERE id=$RID;" >/dev/null 2>&1 || true
  local left
  left=$(sqlexec "SELECT count(*) FROM verifications WHERE subject_type='customer' AND subject_id=$CID")
  if [ "${left:-x}" = "0" ] || [ -z "$left" ]; then
    log "OK   清理完成(客户 $CID 无残留核验单)"
  else
    log "WARN 残留核验单 $left 条需人工巡检(客户 $CID)"
  fi
}
trap cleanup EXIT

log "== 实名流程真环境冒烟 开始 BASE=$BASE =="

TOK=$(api POST /api/admin/v1/auth/login '{"username":"admin","password":"admin123"}' | jq -r '.data.token // empty')
[ -n "$TOK" ] || { log "FATAL 登录失败,中止"; exit 1; }
log "OK   admin 登录取得 token"

# 造数引用:取一条三键齐全的既有客户行(0/NULL 过 RequirePositiveID 会 42200);
# 输出经 tr 清白字符,ssh 回传换行会打穿 JSON 数字位
REFS=$(sqlexec "SELECT legal_entity_id||','||address_id||','||region_id FROM customers
 WHERE coalesce(legal_entity_id,0)>0 AND coalesce(address_id,0)>0 AND coalesce(region_id,0)>0
 ORDER BY id DESC LIMIT 1" | tr -d '[:space:]')
LE=${REFS%%,*}; REST=${REFS#*,}; AD=${REST%%,*}; RG=${REST##*,}
[ -n "${LE:-}" ] || { log "FATAL 库内无参照客户行,无法安全造数"; exit 1; }

TAG="acc_rn_$(date +%s)"
IDNO="11010119900101$((RANDOM % 90 + 10))"
PHONE="139$(printf '%07d' $((RANDOM * RANDOM % 10000000)))"

BODY=$(api POST /api/user/v1/customer-registrations "{\"name\":\"$TAG\",\"phone\":\"$PHONE\",\"idCardNo\":\"$IDNO\",\"legalEntityId\":$LE,\"addressId\":$AD,\"regionId\":$RG}")
RID=$(echo "$BODY" | jq -r '.data.id // empty')
if [ -z "$RID" ]; then log "DEBUG 注册原始响应: $BODY(REFS=[$REFS] LE=$LE AD=$AD RG=$RG)"; fi
expect "公开注册申请落地" "$([ -n "$RID" ] && echo yes)" "yes"

CID=$(api POST "/api/admin/v1/customer-registrations/$RID/approve" '{}' | jq -r '.data.customerId // empty')
expect "审核通过建客户主档(id>0)" "$([ -n "$CID" ] && [ "$CID" != null ] && [ "$CID" -gt 0 ] 2>/dev/null && echo yes)" "yes"

RES1=$(api POST "/api/admin/v1/customers/$CID/real-name" "{\"realName\":\"$TAG\",\"idCardNo\":\"$IDNO\",\"method\":\"人工\"}" | jq -r '.data.result // empty')
if [ "$RES1" = "PENDING" ]; then
  log "OK   代录落 PENDING(二要素通道未启用,走人工)"
else
  log "NOTE 代录即时判定 result=$RES1(102 已配自动通道);跳过待办断言段,仅验回执/状态链路"
fi

NID=""
if [ "$RES1" = "PENDING" ]; then
  NID=$(api GET '/api/admin/v1/notifications?page=1&pageSize=50' | jq -r ".data.items[] | select(.refType==\"realname\" and .refId==\"customer/$CID\") | select(.resolved|not) | .id" | head -1)
  expect "待办中心出现未办实名待审" "$([ -n "$NID" ] && echo yes)" "yes"

  api POST /api/admin/v1/notifications/read '{"ids":[]}' >/dev/null
  U0=$(api GET /api/admin/v1/notifications/unread-count | jq -r '.data.count')
  expect "已读后未读数归零" "$U0" "0"
fi

R_FAIL=$(api POST "/api/admin/v1/verifications/customer/$CID/verify" '{"result":"FAIL","reason":"e2e-smoke 证件照片模糊"}' | jq -r '.data.result // empty')
expect "后台驳回 FAIL 生效" "$R_FAIL" "FAIL"
expect "驳回后实名状态仍 PENDING" "$(sqlexec "SELECT real_name_status FROM customers WHERE id=$CID")" "PENDING"

MSG1=$(sqlexec "SELECT count(*) FROM portal_messages WHERE customer_id=$CID AND payload->>'title'='实名认证未通过'")
expect "驳回站内回执落库" "${MSG1:-0}" "1"

if [ "$RES1" = "PENDING" ]; then
  RESOLVED1=$(api GET '/api/admin/v1/notifications?page=1&pageSize=50' | jq -r ".data.items[] | select(.id==$NID) | .resolved")
  expect "审核终态办结原待办" "$RESOLVED1" "true"
fi

sleep 1
RES2=$(api POST "/api/admin/v1/customers/$CID/real-name" "{\"realName\":\"$TAG\",\"idCardNo\":\"$IDNO\",\"method\":\"人工\"}" | jq -r '.data.result // empty')
if [ "$RES2" = "PENDING" ]; then
  REVIVE=$(api GET '/api/admin/v1/notifications?page=1&pageSize=50' | jq -r ".data.items[] | select(.refType==\"realname\" and .refId==\"customer/$CID\") | select(.resolved|not) | \"\(.id)|\(.title)\"" | head -1)
  RID2="${REVIVE%%|*}"
  expect "重提复活同一待办(id 不变)" "$RID2" "$NID"
  U1=$(api GET /api/admin/v1/notifications/unread-count | jq -r '.data.count')
  if [ "$U1" -ge 1 ] 2>/dev/null; then PASS_CNT=$((PASS_CNT+1)); log "OK   复活清读回执,未读数回升 count=$U1";
  else FAIL_CNT=$((FAIL_CNT+1)); log "FAIL 复活后未读数仍为 0(读回执未被清)"; fi
else
  log "NOTE 重提即时判定 result=$RES2,跳过复活断言"
fi

R_PASS=$(api POST "/api/admin/v1/verifications/customer/$CID/verify" '{"result":"PASS"}' | jq -r '.data.result // empty')
expect "复核 PASS 生效" "$R_PASS" "PASS"
expect "PASS 同步主档 VERIFIED" "$(sqlexec "SELECT real_name_status FROM customers WHERE id=$CID")" "VERIFIED"
MSG2=$(sqlexec "SELECT count(*) FROM portal_messages WHERE customer_id=$CID AND payload->>'title'='实名认证已通过'")
expect "通过站内回执落库" "${MSG2:-0}" "1"

log "== 结果: PASS=$PASS_CNT FAIL=$FAIL_CNT =="
log "== 明细日志: $LOG =="
[ "$FAIL_CNT" = "0" ]
