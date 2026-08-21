-- 四码 quad_links.asset_id 允许 NULL(预绑定阶段无资产,扫码时再填)。
-- 原 UNIQUE(asset_id) 约束删除,改为 partial unique:仅非空时唯一。
-- 配合 applyTag(环节5)实现:下单→预占端口→落四码 UNLINKED(status 待扫码)。
-- 配合 scanBind(环节9)实现:扫码→EPC 反查资产→回填 asset_id→置 LINKED。
BEGIN;

-- 1. 删除旧 UNIQUE 约束(四码各列各建 UNIQUE,见 000012)。
--    PG 不直接 DROP CONSTRAINT，用 DROP INDEX 触发。
DROP INDEX IF EXISTS quad_links_asset_id_key;
DROP INDEX IF EXISTS quad_links_customer_id_key;
DROP INDEX IF EXISTS quad_links_port_id_key;
DROP INDEX IF EXISTS quad_links_address_id_key;

-- 2. asset_id 改为允许 NULL。
ALTER TABLE quad_links
    ALTER COLUMN asset_id DROP NOT NULL;

-- 3. 建 partial unique 索引:仅非空时唯一(允许 NULL 不插入索引)。
CREATE UNIQUE INDEX uq_quad_links_asset
    ON quad_links (asset_id) WHERE asset_id IS NOT NULL;

CREATE UNIQUE INDEX uq_quad_links_customer
    ON quad_links (customer_id) WHERE customer_id IS NOT NULL;

CREATE UNIQUE INDEX uq_quad_links_port
    ON quad_links (port_id) WHERE port_id IS NOT NULL;

CREATE UNIQUE INDEX uq_quad_links_address
    ON quad_links (address_id) WHERE address_id IS NOT NULL;

COMMIT;