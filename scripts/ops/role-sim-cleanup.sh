#!/usr/bin/env bash
# role-sim-cleanup: 本套件专项造数回收(订单/地址/资源等 acc_ 常规造数由
# acceptance-cleanup.sh 按模式回收, 这里只处理常规模式覆盖不到的实体):
# 投诉工单及 CS 子表 / 客户注册+审核通过所建 customers / 师傅注册+通过所建 workers /
# admin 场景所建业务账号及其 API key。全部单事务删除, 失败即退出非 0。
set -u
SSH_HOST="imeepos@192.168.0.102"

# ids_to_in "1 2 3" -> "1,2,3"; 空 -> "NULL"(保证 IN 合法)。
ids_to_in() {
  if [ -z "$(printf '%s' "$1" | tr -d '[:space:]')" ]; then
    echo "NULL"
  else
    printf '%s' "$1" | tr -s ' ' ',' | sed 's/,$//'
  fi
}

# cleanup_special <ticket_no> <reg_ids> <wreg_ids> <account_id>
cleanup_special() {
  local ticket="$1" reg_ids="$2" wreg_ids="$3" account_id="$4" reg_in wreg_in
  reg_in=$(ids_to_in "$reg_ids")
  wreg_in=$(ids_to_in "$wreg_ids")
  if [ -z "$account_id" ]; then
    account_id="NULL"
  fi
  ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST"     "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q -tA" <<SQL
BEGIN;
CREATE TEMP TABLE sim_regs AS SELECT id FROM customer_registrations WHERE id IN ($reg_in);
CREATE TEMP TABLE sim_wregs AS SELECT id FROM worker_registrations WHERE id IN ($wreg_in);
CREATE TEMP TABLE sim_cps AS SELECT id FROM complaints WHERE ticket_no='$ticket';
DELETE FROM cs_callbacks WHERE ticket_id IN (SELECT id FROM sim_cps);
DELETE FROM cs_ticket_events WHERE ticket_id IN (SELECT id FROM sim_cps);
DELETE FROM cs_ticket_extensions WHERE ticket_id IN (SELECT id FROM sim_cps);
DELETE FROM complaints WHERE id IN (SELECT id FROM sim_cps);
DELETE FROM customers WHERE id IN (SELECT customer_id FROM customer_registrations WHERE id IN (SELECT id FROM sim_regs) AND customer_id IS NOT NULL);
DELETE FROM customer_registrations WHERE id IN (SELECT id FROM sim_regs);
DELETE FROM workers WHERE id IN (SELECT worker_id FROM worker_registrations WHERE id IN (SELECT id FROM sim_wregs) AND worker_id IS NOT NULL);
DELETE FROM worker_registrations WHERE id IN (SELECT id FROM sim_wregs);
DELETE FROM api_keys WHERE subject_type='account' AND subject_ref='$account_id';
DELETE FROM accounts WHERE id='$account_id';
COMMIT;
SQL
}

# risk_guard_off/restore: 直营风控与验收共存(同客户高频下单会被 phoneCap 拦)。
RISK_WAS_ON=0  # 直营风控与验收共存(同客户高频下单会被 phoneCap 拦)
risk_guard_off() {
  if curl -sS -m 10 "$API/params/risk.direct.enabled" -H "X-API-Key: $K_ADMIN" 2>/dev/null | grep -q true; then
    curl -sS -m 10 -X PUT "$API/params/risk.direct.enabled" -H "X-API-Key: $K_ADMIN" -H "Content-Type: application/json" -d '{"value":"false"}' >/dev/null
    RISK_WAS_ON=1
    echo "[risk] 直营风控已临时关停(收官恢复)" >&2
  fi
}
risk_guard_restore() {
  if [ "$RISK_WAS_ON" = "1" ]; then
    curl -sS -m 10 -X PUT "$API/params/risk.direct.enabled" -H "X-API-Key: $K_ADMIN" -H "Content-Type: application/json" -d '{"value":"true"}' >/dev/null
    echo "[risk] 直营风控已恢复" >&2
  fi
}




# cleanup_rls_patterns: RLS 前缀造数回收(角色模拟地址/资源/端口/标签/资产/模板)。
cleanup_rls_patterns() {
	ssh -o ConnectTimeout=10 -o BatchMode=yes "$SSH_HOST" "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -v ON_ERROR_STOP=1 -q" <<'SQL'
BEGIN;
CREATE TEMP TABLE rls_addr AS SELECT id FROM addresses WHERE name LIKE '角色模拟-%';
CREATE TEMP TABLE rls_orders AS SELECT id FROM orders WHERE address_id IN (SELECT id FROM rls_addr);
CREATE TEMP TABLE rls_ports AS SELECT id FROM ports WHERE order_id IN (SELECT id FROM rls_orders) OR port_code LIKE 'P-RLS-%';
CREATE TEMP TABLE rls_complaints AS SELECT id FROM complaints WHERE order_id IN (SELECT id FROM rls_orders);
DELETE FROM scan_logs WHERE order_id IN (SELECT id FROM rls_orders);
DELETE FROM dispatch_tickets WHERE order_id IN (SELECT id FROM rls_orders);
DELETE FROM order_stages WHERE order_id IN (SELECT id FROM rls_orders);
DELETE FROM activation_callbacks WHERE order_id IN (SELECT id FROM rls_orders);
DELETE FROM provision_logs WHERE task_id IN (SELECT id FROM provision_tasks WHERE order_id IN (SELECT id FROM rls_orders));
DELETE FROM provision_tasks WHERE order_id IN (SELECT id FROM rls_orders);
DELETE FROM reserve_records WHERE order_id IN (SELECT id FROM rls_orders) OR port_id IN (SELECT id FROM rls_ports);
DELETE FROM port_change_history WHERE port_id IN (SELECT id FROM rls_ports);
DELETE FROM quad_links WHERE address_id IN (SELECT id FROM rls_addr) OR port_id IN (SELECT id FROM rls_ports);
DELETE FROM dismantles WHERE order_id IN (SELECT id FROM rls_orders);
DELETE FROM install_logs WHERE order_id IN (SELECT id FROM rls_orders);
DELETE FROM partner_commission_ledger WHERE order_id IN (SELECT id FROM rls_orders);
DELETE FROM cs_callbacks WHERE ticket_id IN (SELECT id FROM complaints WHERE order_id IN (SELECT id FROM rls_orders));
DELETE FROM cs_ticket_events WHERE ticket_id IN (SELECT id FROM complaints WHERE order_id IN (SELECT id FROM rls_orders));
DELETE FROM cs_ticket_extensions WHERE ticket_id IN (SELECT id FROM complaints WHERE order_id IN (SELECT id FROM rls_orders));
DELETE FROM complaints WHERE order_id IN (SELECT id FROM rls_orders);
DELETE FROM orders WHERE id IN (SELECT id FROM rls_orders);
DELETE FROM ports WHERE id IN (SELECT id FROM rls_ports);
DELETE FROM asset_assignments WHERE asset_id IN (SELECT id FROM assets WHERE asset_code LIKE 'A-RLS-%');
DELETE FROM asset_lifecycles WHERE asset_id IN (SELECT id FROM assets WHERE asset_code LIKE 'A-RLS-%');
DELETE FROM worker_replace_logs WHERE old_tag_id IN (SELECT id FROM tags WHERE tag_no LIKE 'T-RLS-%' OR epc_code LIKE 'EPC-RLS-%')
  OR new_tag_id IN (SELECT id FROM tags WHERE tag_no LIKE 'T-RLS-%' OR epc_code LIKE 'EPC-RLS-%');
DELETE FROM assets WHERE asset_code LIKE 'A-RLS-%';
DELETE FROM tags WHERE tag_no LIKE 'T-RLS-%' OR epc_code LIKE 'EPC-RLS-%';
DELETE FROM asset_batches WHERE code LIKE 'RK-RLS-%';
DELETE FROM pon_onu_alloc WHERE olt_resource_id IN (SELECT id FROM resources WHERE code LIKE 'OLT-RLS-%');
DELETE FROM resources WHERE code LIKE 'OLT-RLS-%' OR code LIKE 'SPL-RLS-%';
DELETE FROM provision_templates WHERE code LIKE 'TPL-RLS-%';
DELETE FROM customers WHERE address_id IN (SELECT id FROM rls_addr) OR id IN (SELECT customer_id FROM customer_registrations WHERE source='acc_sim' AND customer_id IS NOT NULL);
DELETE FROM customer_registrations WHERE source='acc_sim' OR address_id IN (SELECT id FROM rls_addr);
DELETE FROM workers WHERE id IN (SELECT worker_id FROM worker_registrations WHERE review_note LIKE 'acc_%' AND worker_id IS NOT NULL);
DELETE FROM worker_registrations WHERE review_note LIKE 'acc_%' OR phone LIKE '1382%';
DELETE FROM addresses WHERE id IN (SELECT id FROM rls_addr);
COMMIT;
SQL
}

