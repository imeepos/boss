-- 000163 down:严格逆序回滚
DROP TABLE IF EXISTS procurement_receipts;
DROP TABLE IF EXISTS procurement_order_items;
DROP TABLE IF EXISTS procurement_orders;
DROP TABLE IF EXISTS procurement_suppliers;
ALTER TABLE asset_batches DROP COLUMN IF EXISTS warehouse_lng;
ALTER TABLE asset_batches DROP COLUMN IF EXISTS warehouse_lat;
