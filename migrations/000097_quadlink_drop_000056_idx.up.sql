-- 清理 000056 时代残留的 *_active 部分唯一索引。
-- 契约裁定(fields.md §5.1 / terms.md §5 Amended 2026-08-21):
--   000088 起 customer 可 1:N(一客户多链路),customer_id 不再唯一;
--   asset/port/address 各保持至多一条非空活跃链路(000086 非空唯一索引即权威)。
-- 但 000086/000088 只删了 000086 的非空唯一索引与原始 UNIQUE 约束,
-- 漏删 000056 的四个 _active 索引:
--   - uq_quad_links_customer_active 直接违反 1:N 契约(复购客户第二单扫码必 23505);
--   - asset/port/address 三列 _active 被 000086 非空唯一完全覆盖,纯冗余。
-- 复现: 客户已有 LINKED 链路后新订单 scan-bind →
--   "duplicate key ... uq_quad_links_customer_active (SQLSTATE 23505)"。
BEGIN;

DROP INDEX IF EXISTS uq_quad_links_customer_active;
DROP INDEX IF EXISTS uq_quad_links_asset_active;
DROP INDEX IF EXISTS uq_quad_links_port_active;
DROP INDEX IF EXISTS uq_quad_links_address_active;

COMMIT;
