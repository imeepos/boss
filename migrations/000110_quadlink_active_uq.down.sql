-- 回滚 000110:恢复 000086 形态的非空唯一(不区分状态)。
-- 注意:若已产生"同码多行 UNLINKED",回滚会因重复值失败,需先清理历史行。
BEGIN;

DROP INDEX IF EXISTS uq_quad_links_asset;
DROP INDEX IF EXISTS uq_quad_links_port;
DROP INDEX IF EXISTS uq_quad_links_address;

CREATE UNIQUE INDEX uq_quad_links_asset
    ON quad_links (asset_id) WHERE asset_id IS NOT NULL;
CREATE UNIQUE INDEX uq_quad_links_port
    ON quad_links (port_id) WHERE port_id IS NOT NULL;
CREATE UNIQUE INDEX uq_quad_links_address
    ON quad_links (address_id) WHERE address_id IS NOT NULL;

COMMIT;
