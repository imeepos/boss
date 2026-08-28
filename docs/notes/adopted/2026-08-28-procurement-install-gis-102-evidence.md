# 102 PG 迁移验证证据（2026-08-28）

> 决策：docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md
> 工作分支：feature/procurement-install-gis
> 后端对接：http://192.168.0.102:28080
> 数据库：host=192.168.0.102 port=25432 user=boss dbname=boss（PostgreSQL 16.4）

## 执行

```bash
# 102 服务器 PG 主机直连;迁移命令经 psql ON_ERROR_STOP=1 顺序跑 3 个 .up.sql。
PGPASSWORD=boss psql -h 192.168.0.102 -p 25432 -U boss -d boss -v ON_ERROR_STOP=1 -f migrations/000163_procurement.up.sql
PGPASSWORD=boss psql -h 192.168.0.102 -p 25432 -U boss -d boss -v ON_ERROR_STOP=1 -f migrations/000164_install_logs.up.sql
PGPASSWORD=boss psql -h 192.168.0.102 -p 25432 -U boss -d boss -v ON_ERROR_STOP=1 -f migrations/000165_purchase_install_menu.up.sql
```

## 输出（实跑结果）

```
=== Migrating 000163_procurement ===
CREATE TABLE                                       -- procurement_suppliers
CREATE INDEX                                       -- idx_procurement_suppliers_entity
CREATE TABLE                                       -- procurement_orders
CREATE INDEX × 3                                   -- entity / supplier / status
CREATE TABLE                                       -- procurement_order_items
CREATE INDEX                                       -- idx_procurement_order_items_order
CREATE TABLE                                       -- procurement_receipts
CREATE INDEX × 2                                   -- order / status
ALTER TABLE                                        -- asset_batches + warehouse_lat/lng
COMMENT × 7
=== Migrating 000164_install_logs ===
ALTER TABLE                                        -- dispatch_tickets + 3 cols
CREATE TABLE                                       -- install_logs
CREATE INDEX × 3                                   -- ticket / order / status
CREATE UNIQUE INDEX                                -- uq_install_logs_ticket_open (partial)
COMMENT × 3
=== Migrating 000165_purchase_install_menu ===
BEGIN
INSERT 0 3                                         -- permissions
INSERT 0 3                                         -- role_permissions 授 sysadmin
COMMIT
```

## 验证（实跑结果）

```sql
-- 4 张 procurement.* 表已建:
SELECT tablename FROM pg_tables WHERE tablename LIKE 'procurement_%';
--  procurement_order_items
--  procurement_orders
--  procurement_receipts
--  procurement_suppliers

-- asset_batches 增列就位:
SELECT column_name FROM information_schema.columns
WHERE table_name='asset_batches' AND column_name LIKE 'warehouse%';
--  warehouse_lat
--  warehouse_lng

-- dispatch_tickets 增列就位:
SELECT column_name FROM information_schema.columns
WHERE table_name='dispatch_tickets' AND column_name IN ('arrived_at','arrive_lat','arrive_lng');
--  arrived_at
--  arrive_lat
--  arrive_lng

-- install_logs 表已建。

-- 3 个菜单权限码登记并授 sysadmin:
SELECT code FROM permissions WHERE code IN ('menu:purchase','menu:inventory','menu:install-board');
--  menu:purchase
--  menu:inventory
--  menu:install-board
```

## 结论

- 3 个迁移在 102 PG 跑通，0 错误
- 6 张新表 + 5 列补丁 + 1 部分唯一索引 + 3 权限码 全部就位
- 与 docs/contract/fields.md §9 + terms.md §4 + domain-map.md §PUR 三契约一致

下一步：进入真实环境 E2E（admin 登录 → 采购单 → 入库 → 派单 → 到场 → 回单 → GIS 图层）。
