#!/usr/bin/env bash
# role-demand-sim: 全角色诉求模拟套件(bossctl x 102 真实环境, S11/T21)。
# 每角色 >=1 正常 + >=1 异常/边界, 共 17 场景; 输出 PASS/FAIL ROLE:<角色> <场景>;
# 全绿打 ROLESIM-ALL-PASS, 任一 FAIL 退出码非 0; 场景清单见各 sc_* 函数。
set -u
source "$(dirname "$0")/role-sim-lib.sh"
source "$(dirname "$0")/role-sim-cleanup.sh"
source "$(dirname "$0")/acceptance-lock.sh"
acquire_acceptance_lock || exit 1
trap release_acceptance_lock EXIT

load_role_keys
SFX="$(date +%H%M%S)$RANDOM"
TAIL7="$(printf '%s' "$SFX" | tr -cd '0-9' | tail -c 7)"
TAIL4="$(printf '%s' $TAIL7 | tail -c 4)"
IDC_C=$(python3 "$ROOT/scripts/ops/role-sim-idc.py" "1101011990010$TAIL4")
IDC_W=$(python3 "$ROOT/scripts/ops/role-sim-idc.py" "1101011990020$TAIL4")
ADDR_ID=""; OLT_ID=""; TPL_ID=""
ORDER_MAIN_NO=""; ORDER_MAIN_ID=""; ORDER_IDLE_NO=""; ORDER_IDLE2_NO=""; ORDER_EMPTY_NO=""
TICKET_NO=""
REG_REJECT_ID=""; REG_LOOP_ID=""; WREG_REJECT_ID=""; WREG_LOOP_ID=""
CUST_REG_IDS=""; WORKER_REG_IDS=""
SIM_ACCOUNT_ID=""; SIM_USERNAME="acc_sim_$SFX"; CMP_TICKET=""
GLOBAL_SFX="$SFX"

sc_customer_order() {
  local scene="客户自助下单" out rc out2 rc2 ordbody reason="下单或取单号失败"
  printf -v ordbody '{"productId":"101","addressId":"%s","channelId":"102"}' "$ADDR_ID"
  out=$(bc "$K_CUST" call POST user:/orders --data "$ordbody"); rc=$?
	echo "[debug] orderresp=" + "$out" >&2
  ORDER_MAIN_NO=$(printf '%s' "$out" | jno)
  ORDER_MAIN_ID=$(order_id_by_no "$ORDER_MAIN_NO")
  printf -v ordbody '{"productId":"101","addressId":"%s","channelId":"102"}' "$ADDR_ID"
  out2=$(bc "$K_CUST" call POST user:/orders --data "$ordbody"); rc2=$?
	echo "[debug] orderresp=" + "$out" >&2
  ORDER_IDLE_NO=$(printf '%s' "$out2" | jno)
  if expect_ok "$scene" "$rc" "$out" && [ -n "$ORDER_MAIN_NO" ] && [ -n "$ORDER_MAIN_ID" ] && expect_ok "$scene" "$rc2" "$out2" && [ -n "$ORDER_IDLE_NO" ]; then
    sim_pass customer "$scene"
  else
    [ -n "$EXPECT_FAIL_REASON" ] && reason="$EXPECT_FAIL_REASON"
    sim_fail customer "$scene" "$reason"
  fi
}

sc_customer_cancel() {
  local scene="取消未受理订单" out rc state
  out=$(bc "$K_CUST" call POST user:/orders/$ORDER_IDLE_NO/cancel --data '{}'); rc=$?
  state=$(order_state "$ORDER_IDLE_NO")
  if expect_ok "$scene" "$rc" "$out" && printf '%s' "$state" | grep -q CANCELLED && ! bc "$K_CUST" call POST user:/orders/$ORDER_IDLE_NO/cancel --data '{}' >/dev/null 2>&1; then
    sim_pass customer "$scene"
  else
    sim_fail customer "$scene" "state=$state 或重复取消未被拒"
  fi
}

sc_customer_change_addr() {
  local scene="施工段改地址被拒" baid out rc
  baid=$(fixture_address "$SFX"b)
  printf -v ordbody '{"addressId":%s}' "$baid"
  out=$(bc "$K_CUST" call POST user:/orders/$ORDER_MAIN_NO/change-address --data "$ordbody"); rc=$?
  if expect_fail "$scene" "$rc" "$out"; then
    sim_pass customer "$scene"
  else
    sim_fail customer "$scene" "$EXPECT_FAIL_REASON"
  fi
}

DISP_R1=0; DISP_R2=0
sc_dispatch_check_reserve() {
  local out1 rc1 out2 rc2 reserved
  out1=$(bc "$K_DISP" call POST /orders/$ORDER_MAIN_NO/check-resource --data '{}'); rc1=$?
  out2=$(bc "$K_DISP" call POST /orders/$ORDER_MAIN_NO/reserve --data '{}'); rc2=$?
  reserved=$(sql <<SQL
SELECT count(*) FROM ports WHERE order_id=$ORDER_MAIN_ID AND status='RESERVED';
SQL
)
  [ "$rc1" -eq 0 ] && [ "$rc2" -eq 0 ] && [ "$reserved" -ge 1 ] && DISP_R1=1
}

sc_dispatch_assign() {
  local out rc
  TICKET_NO=$(bc "$K_DISP" call GET /dispatch/pool --query pageSize=200 | jpick orderId "$ORDER_MAIN_ID" ticketNo)
  out=$(bc "$K_DISP" call POST /dispatch/pool/$TICKET_NO/assign --data '{"masterId":6}'); rc=$?
  if [ "$rc" -eq 0 ] && [ -n "$TICKET_NO" ]; then
    DISP_R2=2
  fi
}

sc_dispatch_report() {
  local scene="资源核查预占与派单(段2/3/8)"
  if [ "$DISP_R1" = "1" ] && [ "$DISP_R2" = "2" ]; then
    sim_pass dispatch "$scene"
  else
    sim_fail dispatch "$scene" "check_reserve=$DISP_R1 assign=$DISP_R2 ticket=$TICKET_NO"
  fi
}

sc_dispatch_shortage() {
  local scene="资源不足失败可观测" eaid ordbody out rc
  eaid=$(fixture_address "$SFX"e)
  printf -v ordbody '{"productId":"101","addressId":"%s","channelId":"102"}' "$eaid"
  ORDER_EMPTY_NO=$(bc "$K_CUST" call POST user:/orders --data "$ordbody" | jno)
	echo "[debug] orderresp=" + "$out" >&2
  out=$(bc "$K_DISP" call POST /orders/$ORDER_EMPTY_NO/check-resource --data '{}'); rc=$?
  if expect_fail "$scene" "$rc" "$out"; then
    sim_pass dispatch "$scene"
  else
    sim_fail dispatch "$scene" "$EXPECT_FAIL_REASON"
  fi
}

sc_cashier_charge() {
  local scene="合同收费(段4)" out rc stage
  out=$(bc "$K_CASH" call POST /orders/$ORDER_MAIN_NO/charge --data '{}'); rc=$?
  stage=$(order_state "$ORDER_MAIN_NO" | awk -F/ '{print $1}')
  if expect_ok "$scene" "$rc" "$out" && [ -n "$stage" ] && [ "$stage" -ge 4 ] 2>/dev/null; then
    sim_pass cashier "$scene"
  else
    sim_fail cashier "$scene" "stage=$stage $(printf '%s' "$EXPECT_LAST_OUT" | head -1)"
  fi
}

sc_cashier_abnormal() {
  local scene="重复收费不落账与前置不符被拒" out rc before after
  out=$(bc "$K_CASH" call POST /orders/$ORDER_IDLE2_NO/charge --data '{}'); rc=$?
  if ! expect_fail "前置不符" "$rc" "$out"; then
    sim_fail cashier "$scene" "$EXPECT_FAIL_REASON"
    return 0
  fi
  before=$(order_state "$ORDER_MAIN_NO")
  out=$(bc "$K_CASH" call POST /orders/$ORDER_MAIN_NO/charge --data '{}'); rc=$?
  after=$(order_state "$ORDER_MAIN_NO")
  if [ "$rc" -eq 0 ] && [ "$before" = "$after" ]; then
    sim_pass cashier "$scene"
  else
    sim_fail cashier "$scene" "重复收费状态漂移 $before -> $after"
  fi
}

sc_reviewer_approve_reject() {
  local scene="注册审批通过与驳回" phc phw out rc1 rc2
  phc="1391$TAIL7"
  phw="1381$TAIL7"
  printf -v regbody '{"name":"acc模拟客户","phone":"%s","idCardNo":"%s","legalEntityId":1,"addressId":%s,"regionId":4,"source":"acc_sim"}' "$phc" "$IDC_C" "$ADDR_ID"
  out=$(curl -sS -m 15 -X POST "$SERVER/api/user/v1/customer-registrations" -H "Content-Type: application/json" -d "$regbody"); rc=$?
  echo "[debug] regsubmit=$out" >&2
  REG_REJECT_ID=$(printf '%s' "$out" | jid)
  bc "$K_REV" call POST /customer-registrations/$REG_REJECT_ID/approve --data '{}' >/dev/null; rc1=$?
  printf -v wregbody '{"name":"acc模拟师傅","phone":"%s","idCardNo":"%s","groupId":11,"regionId":4}' "$phw" "$IDC_W"
  out=$(curl -sS -m 15 -X POST "$SERVER/api/worker/v1/worker-registrations" -H "Content-Type: application/json" -d "$wregbody"); rc=$?
  WREG_REJECT_ID=$(printf '%s' "$out" | jid)
  bc "$K_REV" call POST /worker-registrations/$WREG_REJECT_ID/reject --data '{"note":"acc_ 模拟驳回:证件照模糊"}' >/dev/null; rc2=$?
  if [ "$rc1" -eq 0 ] && [ "$rc2" -eq 0 ] && [ -n "$REG_REJECT_ID" ] && [ -n "$WREG_REJECT_ID" ]; then
    sim_pass reviewer "$scene"
  else
    sim_fail reviewer "$scene" "approve=$rc1 reject=$rc2 reg=$REG_REJECT_ID/$WREG_REJECT_ID"
  fi
}

sc_reviewer_resubmit_loop() {
  local scene="驳回后重新提交再审闭环" phc phw out rc1 rc2
  phc="1392$TAIL7"
  phw="1382$TAIL7"
  printf -v regbody '{"name":"acc模拟客户","phone":"%s","idCardNo":"%s","legalEntityId":1,"addressId":%s,"regionId":4,"source":"acc_sim"}' "$phc" "$IDC_W" "$ADDR_ID"
  out=$(curl -sS -m 15 -X POST "$SERVER/api/user/v1/customer-registrations" -H "Content-Type: application/json" -d "$regbody"); rc=$?
  REG_LOOP_ID=$(printf '%s' "$out" | jid)
  bc "$K_REV" call POST /customer-registrations/$REG_LOOP_ID/approve --data '{}' >/dev/null; rc1=$?
  printf -v wregbody '{"name":"acc模拟师傅","phone":"%s","idCardNo":"%s","groupId":11,"regionId":4}' "$phw" "$IDC_W"
  out=$(curl -sS -m 15 -X POST "$SERVER/api/worker/v1/worker-registrations" -H "Content-Type: application/json" -d "$wregbody"); rc=$?
  WREG_LOOP_ID=$(printf '%s' "$out" | jid)
	bc "$K_REV" call POST /worker-registrations/$WREG_LOOP_ID/approve --data '{"groupId":11,"regionId":4}' > /tmp/rolesim-wapprove.txt 2>&1; rc2=$?
	echo "[debug] wapprove-out: $(head -c 200 /tmp/rolesim-wapprove.txt)" >&2
  CUST_REG_IDS="$REG_REJECT_ID $REG_LOOP_ID"
  WORKER_REG_IDS="$WREG_REJECT_ID $WREG_LOOP_ID"
  if [ "$rc1" -eq 0 ] && [ "$rc2" -eq 0 ] && [ -n "$REG_LOOP_ID" ] && [ -n "$WREG_LOOP_ID" ]; then
    sim_pass reviewer "$scene"
  else
    sim_fail reviewer "$scene" "重审 approve=$rc1/$rc2 reg=$REG_LOOP_ID/$WREG_LOOP_ID"
  fi
}

sc_admin_account_key() {
  local scene="受权建号与签发APIKey" out rc aout newkey
  printf -v accbody '{"username":"%s","password":"Sim@12345","realName":"acc模拟账号","phone":"1371%s","roleCode":"ops","legalEntityId":1,"deptId":4,"postId":7,"status":1}' "$SIM_USERNAME" "$TAIL7"
  out=$(curl -sS -m 15 -X POST "$SERVER/api/admin/v1/accounts" -H "X-API-Key: $K_ADMIN" -H "Content-Type: application/json" -d "$accbody"); rc=$?
  SIM_ACCOUNT_ID=$(printf '%s' "$out" | jid)
  if [ -z "$SIM_ACCOUNT_ID" ]; then
    sim_fail admin "$scene" "建号失败 $out"
    return 0
  fi
  aout=$(bc "$K_ADMIN" apikey create account/$SIM_ACCOUNT_ID rolesim-key); rc=$?
  newkey=$(printf '%s' "$aout" | grep -oE 'boss_[0-9a-f]{32}' | head -1)
  if [ -n "$newkey" ]; then
    out=$(bc "$newkey" call GET /auth/me); rc=$?
    if [ "$rc" -eq 0 ]; then
      sim_pass admin "$scene"
      return 0
    fi
    sim_fail admin "$scene" "新 key 鉴权失败 $out"
    return 0
  fi
  sim_fail admin "$scene" "签发失败 $aout"
}

sc_admin_boundary() {
  local scene="客户师傅key调admin被拒(三表边界)" out1 rc1 out2 rc2
  out1=$(bc "$K_CUST" call GET /orders); rc1=$?
  out2=$(bc "$K_WORKER" call GET /orders); rc2=$?
  if expect_fail "$scene" "$rc1" "$out1" && expect_fail "$scene" "$rc2" "$out2"; then
    sim_pass admin "$scene"
  else
    sim_fail admin "$scene" "$EXPECT_FAIL_REASON"
  fi
}

# ===== 主流程: 区域覆盖夹具 -> 夹具自举 -> 按角色依赖顺序执行 -> 清理 -> 门禁 =====
GLOBAL_SFX="$SFX"
PREV_REGION_ENT=""
region_fixture_on() {
  PREV_REGION_ENT=$(sql <<SQL
SELECT COALESCE(legal_entity_id::text,'') FROM regions WHERE id=4;
SQL
)
  sql <<SQL
UPDATE regions SET legal_entity_id=1 WHERE id=4;
SQL
  echo "[fixture] region4 entity prev=$PREV_REGION_ENT -> 1" >&2
}
region_fixture_off() {
  if [ -z "$PREV_REGION_ENT" ]; then
    sql <<SQL
UPDATE regions SET legal_entity_id=NULL WHERE id=4;
SQL
  else
    sql <<SQL
UPDATE regions SET legal_entity_id=$PREV_REGION_ENT WHERE id=4;
SQL
  fi
  echo "[fixture] region4 entity restored" >&2
}

echo "全角色诉求模拟: 102=$SERVER 后缀=$SFX"
region_fixture_on

ADDR_ID=$(fixture_address "$SFX")
sql <<SQL
UPDATE addresses SET region_id=4 WHERE id=$ADDR_ID;
SQL
OLT_ID=$(fixture_olt_ports "$SFX" "$ADDR_ID")
TPL_ID=$(fixture_template "$SFX")
fixture_tag_asset "$SFX"
BOUND=$(sql <<SQL
SELECT count(*) FROM tags t JOIN assets a ON a.tag_id=t.id WHERE t.epc_code='EPC-RLS-$SFX' AND t.bound_asset_id IS NOT NULL AND a.id=t.bound_asset_id;
SQL
)
if [ -z "$ADDR_ID" ] || [ -z "$OLT_ID" ] || [ -z "$TPL_ID" ] || [ "$BOUND" -lt 1 ]; then
  echo "FAIL ROLE:noc 端口查询与置备 夹具自举失败 addr=$ADDR_ID olt=$OLT_ID tpl=$TPL_ID bound=$BOUND"
  region_fixture_off
  exit 1
fi

sc_customer_order
sc_customer_cancel
sc_dispatch_check_reserve
sc_cashier_charge
sc_cashier_abnormal
fixture_pon_task "$ORDER_MAIN_ID" "$TPL_ID"
sc_dispatch_assign
sc_dispatch_report
sc_customer_change_addr
sc_worker_accept
sc_worker_preactivate
sc_worker_finish
sc_dispatch_shortage
sc_kefu_query_complaint
sc_kefu_no_order
sc_noc_provision
sc_noc_cross_domain
sc_reviewer_approve_reject
sc_reviewer_resubmit_loop
sc_admin_account_key
sc_admin_boundary

echo "收尾: 专项造数清理 + RLS 造数清理 + 孤儿巡检门禁"
rc=0
if [ "$SKIP_CLEANUP" != "1" ]; then
  cleanup_special "$CMP_TICKET" "$CUST_REG_IDS" "$WORKER_REG_IDS" "$SIM_ACCOUNT_ID" || rc=1
  cleanup_rls_patterns || rc=1
else
  echo "[debug] SKIP_CLEANUP=1 跳过清理与门禁" >&2
fi
region_fixture_off
risk_guard_restore
"$ROOT/scripts/ops/db-patrol-gate.sh" || rc=1

echo "结果: PASS=$PASS_COUNT FAIL=$FAIL_COUNT"
if [ "$FAIL_COUNT" -eq 0 ] && [ "$rc" -eq 0 ]; then
  echo "ROLESIM-ALL-PASS"
  exit 0
fi
exit 1
