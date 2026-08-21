BEGIN;

DROP INDEX IF EXISTS uq_quad_links_asset;
DROP INDEX IF EXISTS uq_quad_links_customer;
DROP INDEX IF EXISTS uq_quad_links_port;
DROP INDEX IF EXISTS uq_quad_links_address;

-- 恢复 asset_id NOT NULL。
UPDATE quad_links SET asset_id = 0 WHERE asset_id IS NULL;
ALTER TABLE quad_links
    ALTER COLUMN asset_id SET NOT NULL;

-- 恢复 UNIQUE 约束(PG 自动建索引)。
ALTER TABLE quad_links ADD CONSTRAINT quad_links_asset_id_key UNIQUE (asset_id);
ALTER TABLE quad_links ADD CONSTRAINT quad_links_customer_id_key UNIQUE (customer_id);
ALTER TABLE quad_links ADD CONSTRAINT quad_links_port_id_key UNIQUE (port_id);
ALTER TABLE quad_links ADD CONSTRAINT quad_links_address_id_key UNIQUE (address_id);

COMMIT;