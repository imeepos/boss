-- e2e-asset-linkage-residue.sql -- 装机联动 E2E 收尾残留门禁(A3)。
-- 每行一类 acc_ 造数残留计数,执行后逐类断言为零;口径对齐 acceptance-cleanup.sh。
SELECT 'addr' AS cls, count(*) FROM addresses WHERE name LIKE '验收地址-%'
UNION ALL SELECT 'orders', count(*) FROM orders o WHERE o.address_id IN (SELECT id FROM addresses WHERE name LIKE '验收地址-%')
UNION ALL SELECT 'order_stages', count(*) FROM order_stages s WHERE s.order_id IN (SELECT o.id FROM orders o WHERE o.address_id IN (SELECT id FROM addresses WHERE name LIKE '验收地址-%'))
UNION ALL SELECT 'scan_logs', count(*) FROM scan_logs l WHERE l.order_id IN (SELECT o.id FROM orders o WHERE o.address_id IN (SELECT id FROM addresses WHERE name LIKE '验收地址-%'))
UNION ALL SELECT 'tickets', count(*) FROM dispatch_tickets t WHERE t.order_id IN (SELECT o.id FROM orders o WHERE o.address_id IN (SELECT id FROM addresses WHERE name LIKE '验收地址-%'))
UNION ALL SELECT 'dismantles', count(*) FROM dismantles d WHERE d.order_id IN (SELECT o.id FROM orders o WHERE o.address_id IN (SELECT id FROM addresses WHERE name LIKE '验收地址-%'))
UNION ALL SELECT 'prov_tasks', count(*) FROM provision_tasks t WHERE t.order_id IN (SELECT o.id FROM orders o WHERE o.address_id IN (SELECT id FROM addresses WHERE name LIKE '验收地址-%'))
UNION ALL SELECT 'quad_links', count(*) FROM quad_links q WHERE q.address_id IN (SELECT id FROM addresses WHERE name LIKE '验收地址-%')
UNION ALL SELECT 'ports', count(*) FROM ports WHERE port_code LIKE 'P-ACC-%'
UNION ALL SELECT 'reserve_rec', count(*) FROM reserve_records r WHERE r.port_id IN (SELECT id FROM ports WHERE port_code LIKE 'P-ACC-%')
UNION ALL SELECT 'assets', count(*) FROM assets WHERE asset_code LIKE 'A-ACC-%'
UNION ALL SELECT 'lifecycles', count(*) FROM asset_lifecycles l WHERE l.asset_id IN (SELECT id FROM assets WHERE asset_code LIKE 'A-ACC-%')
UNION ALL SELECT 'assigns', count(*) FROM asset_assignments a WHERE a.asset_id IN (SELECT id FROM assets WHERE asset_code LIKE 'A-ACC-%')
UNION ALL SELECT 'tags', count(*) FROM tags WHERE tag_no LIKE 'T-ACC-%' OR epc_code LIKE 'EPC-ACC-%'
UNION ALL SELECT 'replace_log', count(*) FROM worker_replace_logs r WHERE r.old_tag_id IN (SELECT id FROM tags WHERE tag_no LIKE 'T-ACC-%' OR epc_code LIKE 'EPC-ACC-%') OR r.new_tag_id IN (SELECT id FROM tags WHERE tag_no LIKE 'T-ACC-%' OR epc_code LIKE 'EPC-ACC-%')
UNION ALL SELECT 'batches', count(*) FROM asset_batches WHERE code LIKE 'RK-ACC-%'
UNION ALL SELECT 'resources', count(*) FROM resources WHERE code LIKE 'OLT-ACC-%' OR code LIKE 'SPL-ACC-%'
UNION ALL SELECT 'templates', count(*) FROM provision_templates WHERE code LIKE 'TPL-ACC-%'
UNION ALL SELECT 'pon_alloc', count(*) FROM pon_onu_alloc p WHERE p.olt_resource_id IN (SELECT id FROM resources WHERE code LIKE 'OLT-ACC-%')
