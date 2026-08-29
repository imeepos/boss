#!/usr/bin/env bash
# acceptance-cleanup: 主链路验收(mainchain-acceptance.sh)造数自清理(2026-08-29)。
# 背景: 验收脚本每轮自举 地址/分光器/端口/标签/资产/订单(带 acc_ 标记)且从不回收,
# 102 累积 31 轮造数(31 地址/31 资源/62 端口/31 订单/31 工单/31 四码/31 资产)。
# 契约: 只删带验收标记的行(地址名 '验收地址-%'、资源 SPL-ACC-%、端口 P-ACC-%、
# 资产 A-ACC-%、批次 RK-ACC-%、标签 T-ACC-%、EPC EPC-ACC-% 及其订单链),
# 不触碰真实业务数据;删除前 pg_dump 备份;单事务 ON_ERROR_STOP,失败整体回滚。
# 2026-08-28 补全: orders 的 FK 子表全量(activation_callbacks/complaints 三子表/
# dismantles/install_logs/partner_commission_ledger)与 地址/端口/资产/标签 的
# 二级子表(customer_histories 等)——2026-08-28 实跑撞 activation_callbacks FK 回滚。
# 用法: scripts/ops/acceptance-cleanup.sh [--apply]   # 默认 dry-run 只计数
# 环境: SSH_HOST 可覆盖(缺省 imeepos@192.168.0.102);PSQL 容器 boss-infra-postgres-1。
set -u
MODE="dry-run"
[ "${1:-}" = "--apply" ] && MODE="apply"
SSH_HOST="${SSH_HOST:-imeepos@192.168.0.102}"
PG="docker exec -i boss-infra-postgres-1 psql -U boss -d boss"

# SQL: 先计数报告;--apply 时备份 + 单事务删除(依赖序:订单子链→标签/端口/资产二级→
# 端口/资产/标签/批次/资源→订单→地址)。
COUNTS=$(cat <<'SQL'
SELECT 'orders(acc)' AS item, count(*) FROM orders o
  WHERE o.address_id IN (SELECT id FROM addresses WHERE name LIKE '验收地址-%');
SELECT 'order_stages(acc)', count(*) FROM order_stages s
  WHERE s.order_id IN (SELECT o.id FROM orders o WHERE o.address_id IN
    (SELECT id FROM addresses WHERE name LIKE '验收地址-%'));
SELECT 'scan_logs(acc)', count(*) FROM scan_logs l
  WHERE l.order_id IN (SELECT o.id FROM orders o WHERE o.address_id IN
    (SELECT id FROM addresses WHERE name LIKE '验收地址-%'));
SELECT 'dispatch_tickets(acc)', count(*) FROM dispatch_tickets t
  WHERE t.order_id IN (SELECT o.id FROM orders o WHERE o.address_id IN
    (SELECT id FROM addresses WHERE name LIKE '验收地址-%'));
SELECT 'activation_callbacks(acc)', count(*) FROM activation_callbacks c
  WHERE c.order_id IN (SELECT o.id FROM orders o WHERE o.address_id IN
    (SELECT id FROM addresses WHERE name LIKE '验收地址-%'));
SELECT 'complaints(acc)', count(*) FROM complaints t
  WHERE t.order_id IN (SELECT o.id FROM orders o WHERE o.address_id IN
    (SELECT id FROM addresses WHERE name LIKE '验收地址-%'));
SELECT 'cs_callbacks(acc)', count(*) FROM cs_callbacks c
  WHERE c.ticket_id IN (SELECT t.id FROM complaints t WHERE t.order_id IN
    (SELECT o.id FROM orders o WHERE o.address_id IN
      (SELECT id FROM addresses WHERE name LIKE '验收地址-%')));
SELECT 'cs_ticket_events(acc)', count(*) FROM cs_ticket_events e
  WHERE e.ticket_id IN (SELECT t.id FROM complaints t WHERE t.order_id IN
    (SELECT o.id FROM orders o WHERE o.address_id IN
      (SELECT id FROM addresses WHERE name LIKE '验收地址-%')));
SELECT 'cs_ticket_extensions(acc)', count(*) FROM cs_ticket_extensions x
  WHERE x.ticket_id IN (SELECT t.id FROM complaints t WHERE t.order_id IN
    (SELECT o.id FROM orders o WHERE o.address_id IN
      (SELECT id FROM addresses WHERE name LIKE '验收地址-%')));
SELECT 'dismantles(acc)', count(*) FROM dismantles d
  WHERE d.order_id IN (SELECT o.id FROM orders o WHERE o.address_id IN
    (SELECT id FROM addresses WHERE name LIKE '验收地址-%'));
SELECT 'install_logs(acc)', count(*) FROM install_logs l
  WHERE l.order_id IN (SELECT o.id FROM orders o WHERE o.address_id IN
    (SELECT id FROM addresses WHERE name LIKE '验收地址-%'));
SELECT 'partner_commission(acc)', count(*) FROM partner_commission_ledger l
  WHERE l.order_id IN (SELECT o.id FROM orders o WHERE o.address_id IN
    (SELECT id FROM addresses WHERE name LIKE '验收地址-%'));
SELECT 'quad_links(acc)', count(*) FROM quad_links q
  WHERE q.address_id IN (SELECT id FROM addresses WHERE name LIKE '验收地址-%');
SELECT 'reserve_records(acc)', count(*) FROM reserve_records r
  WHERE r.port_id IN (SELECT id FROM ports WHERE port_code LIKE 'P-ACC-%');
SELECT 'ports(acc)', count(*) FROM ports WHERE port_code LIKE 'P-ACC-%';
SELECT 'port_change_history(acc)', count(*) FROM port_change_history h
  WHERE h.port_id IN (SELECT id FROM ports WHERE port_code LIKE 'P-ACC-%');
SELECT 'assets(acc)', count(*) FROM assets WHERE asset_code LIKE 'A-ACC-%';
SELECT 'asset_assignments(acc)', count(*) FROM asset_assignments a WHERE a.asset_id IN
  (SELECT id FROM assets WHERE asset_code LIKE 'A-ACC-%');
SELECT 'asset_lifecycles(acc)', count(*) FROM asset_lifecycles l WHERE l.asset_id IN
  (SELECT id FROM assets WHERE asset_code LIKE 'A-ACC-%');
SELECT 'tags(acc)', count(*) FROM tags WHERE tag_no LIKE 'T-ACC-%' OR epc_code LIKE 'EPC-ACC-%';
SELECT 'replace_logs(acc)', count(*) FROM worker_replace_logs r WHERE r.old_tag_id IN
  (SELECT id FROM tags WHERE tag_no LIKE 'T-ACC-%' OR epc_code LIKE 'EPC-ACC-%')
  OR r.new_tag_id IN (SELECT id FROM tags WHERE tag_no LIKE 'T-ACC-%' OR epc_code LIKE 'EPC-ACC-%');
SELECT 'batches(acc)', count(*) FROM asset_batches WHERE code LIKE 'RK-ACC-%';
SELECT 'resources(acc)', count(*) FROM resources WHERE code LIKE 'SPL-ACC-%';
SELECT 'addr_cust_sub(acc)', count(*) FROM customer_histories h WHERE h.address_id IN
  (SELECT id FROM addresses WHERE name LIKE '验收地址-%');
SELECT 'cust_registrations(acc)', count(*) FROM customer_registrations g WHERE g.address_id IN
  (SELECT id FROM addresses WHERE name LIKE '验收地址-%');
SELECT 'customers(acc)', count(*) FROM customers c WHERE c.address_id IN
  (SELECT id FROM addresses WHERE name LIKE '验收地址-%');
SELECT 'lo_accounts(acc)', count(*) FROM lo_accounts l WHERE l.customer_id IN
  (SELECT id FROM customers WHERE address_id IN (SELECT id FROM addresses WHERE name LIKE '验收地址-%'));
SELECT 'resource_assignments(acc)', count(*) FROM resource_assignments ra
  WHERE ra.address_id IN (SELECT id FROM addresses WHERE name LIKE '验收地址-%')
  OR ra.resource_id IN (SELECT id FROM resources WHERE code LIKE 'SPL-ACC-%');
SELECT 'addresses(acc)', count(*) FROM addresses WHERE name LIKE '验收地址-%';
SQL
)

echo "== acceptance-cleanup ($MODE) =="
ssh "$SSH_HOST" "$PG -tAc \"$(echo "$COUNTS" | tr '\n' ' ' | sed 's/;/;/g')\"" 2>/dev/null || {
  echo "FAIL: cannot reach DB via $SSH_HOST"; exit 2; }

if [ "$MODE" != "apply" ]; then
  echo "dry-run 完成。确认无误后执行: $0 --apply"
  exit 0
fi

# 备份(整表导出受影响行所在表,先例 2026-08-25 孤儿清理):
STAMP=$(date +%Y%m%d-%H%M%S)
echo "== 备份到 $SSH_HOST:/tmp/boss_acc_backup_$STAMP.sql =="
ssh "$SSH_HOST" "docker exec -i boss-infra-postgres-1 pg_dump -U boss -d boss \
  -t orders -t order_stages -t scan_logs -t dispatch_tickets -t quad_links \
  -t activation_callbacks -t complaints -t cs_callbacks -t cs_ticket_events -t cs_ticket_extensions \
  -t dismantles -t install_logs -t partner_commission_ledger \
  -t reserve_records -t ports -t port_change_history -t assets -t asset_assignments -t asset_lifecycles \
  -t tags -t worker_replace_logs -t asset_batches -t resources -t resource_assignments \
  -t customer_histories -t customer_registrations -t customers -t lo_accounts -t addresses \
  > /tmp/boss_acc_backup_$STAMP.sql" || { echo "FAIL: backup failed, abort"; exit 2; }

# 单事务删除:任一步失败整体回滚(ON_ERROR_STOP)。
ssh "$SSH_HOST" "$PG -v ON_ERROR_STOP=1" <<'SQL'
BEGIN;
CREATE TEMP TABLE acc_orders AS
  SELECT o.id FROM orders o WHERE o.address_id IN
    (SELECT id FROM addresses WHERE name LIKE '验收地址-%');
CREATE TEMP TABLE acc_complaints AS
  SELECT t.id FROM complaints t WHERE t.order_id IN (SELECT id FROM acc_orders);
DELETE FROM scan_logs WHERE order_id IN (SELECT id FROM acc_orders);
DELETE FROM dispatch_tickets WHERE order_id IN (SELECT id FROM acc_orders);
DELETE FROM order_stages WHERE order_id IN (SELECT id FROM acc_orders);
DELETE FROM activation_callbacks WHERE order_id IN (SELECT id FROM acc_orders);
DELETE FROM cs_callbacks WHERE ticket_id IN (SELECT id FROM acc_complaints);
DELETE FROM cs_ticket_events WHERE ticket_id IN (SELECT id FROM acc_complaints);
DELETE FROM cs_ticket_extensions WHERE ticket_id IN (SELECT id FROM acc_complaints);
DELETE FROM complaints WHERE id IN (SELECT id FROM acc_complaints);
DELETE FROM dismantles WHERE order_id IN (SELECT id FROM acc_orders);
DELETE FROM install_logs WHERE order_id IN (SELECT id FROM acc_orders);
DELETE FROM partner_commission_ledger WHERE order_id IN (SELECT id FROM acc_orders);
DELETE FROM quad_links WHERE address_id IN
  (SELECT id FROM addresses WHERE name LIKE '验收地址-%');
DELETE FROM worker_replace_logs WHERE old_tag_id IN
  (SELECT id FROM tags WHERE tag_no LIKE 'T-ACC-%' OR epc_code LIKE 'EPC-ACC-%')
  OR new_tag_id IN (SELECT id FROM tags WHERE tag_no LIKE 'T-ACC-%' OR epc_code LIKE 'EPC-ACC-%');
DELETE FROM port_change_history WHERE port_id IN
  (SELECT id FROM ports WHERE port_code LIKE 'P-ACC-%');
DELETE FROM reserve_records WHERE port_id IN
  (SELECT id FROM ports WHERE port_code LIKE 'P-ACC-%');
DELETE FROM asset_assignments WHERE asset_id IN
  (SELECT id FROM assets WHERE asset_code LIKE 'A-ACC-%');
DELETE FROM asset_lifecycles WHERE asset_id IN
  (SELECT id FROM assets WHERE asset_code LIKE 'A-ACC-%');
DELETE FROM orders WHERE id IN (SELECT id FROM acc_orders);
DELETE FROM ports WHERE port_code LIKE 'P-ACC-%';
DELETE FROM assets WHERE asset_code LIKE 'A-ACC-%';
DELETE FROM tags WHERE tag_no LIKE 'T-ACC-%' OR epc_code LIKE 'EPC-ACC-%';
DELETE FROM asset_batches WHERE code LIKE 'RK-ACC-%';
DELETE FROM resource_assignments WHERE address_id IN
  (SELECT id FROM addresses WHERE name LIKE '验收地址-%')
  OR resource_id IN (SELECT id FROM resources WHERE code LIKE 'SPL-ACC-%');
DELETE FROM customer_histories WHERE address_id IN
  (SELECT id FROM addresses WHERE name LIKE '验收地址-%');
DELETE FROM customer_registrations WHERE address_id IN
  (SELECT id FROM addresses WHERE name LIKE '验收地址-%');
DELETE FROM lo_accounts WHERE customer_id IN
  (SELECT id FROM customers WHERE address_id IN
    (SELECT id FROM addresses WHERE name LIKE '验收地址-%'));
DELETE FROM customers WHERE address_id IN
  (SELECT id FROM addresses WHERE name LIKE '验收地址-%');
DELETE FROM resources WHERE code LIKE 'SPL-ACC-%';
DELETE FROM addresses WHERE name LIKE '验收地址-%';
COMMIT;
SQL
rc=$?
[ $rc -eq 0 ] && echo "清理事务提交成功(备份: /tmp/boss_acc_backup_$STAMP.sql)" || echo "清理失败,事务已回滚(备份仍在)"
exit $rc
