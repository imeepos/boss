-- 回滚: 恢复 000056 的四个 _active 部分唯一索引。
-- 注意 customer 列恢复后 1:N 契约失效(回到 000088 之前的口径)。
BEGIN;

CREATE UNIQUE INDEX IF NOT EXISTS uq_quad_links_customer_active
    ON quad_links (customer_id) WHERE status IN ('LINKED','CONFLICT');
CREATE UNIQUE INDEX IF NOT EXISTS uq_quad_links_asset_active
    ON quad_links (asset_id) WHERE status IN ('LINKED','CONFLICT');
CREATE UNIQUE INDEX IF NOT EXISTS uq_quad_links_port_active
    ON quad_links (port_id) WHERE status IN ('LINKED','CONFLICT');
CREATE UNIQUE INDEX IF NOT EXISTS uq_quad_links_address_active
    ON quad_links (address_id) WHERE status IN ('LINKED','CONFLICT');

COMMIT;
