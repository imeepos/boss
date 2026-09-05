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

# cleanup_acc_patterns: acc_/验收地址 常规造数回收(订单链/端口/资源/标签/资产/模板)。
cleanup_acc_patterns() {
  local root
  root="$(cd "$(dirname "$0")/../.." && pwd)"
  "$root/scripts/ops/acceptance-cleanup.sh" --apply
}
