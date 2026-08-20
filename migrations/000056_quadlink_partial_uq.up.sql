-- D3(db-design-review): quad_links 四列 UNIQUE 改部分唯一索引。
-- 仅 LINKED/CONFLICT 态占用唯一槽位;UNLINKED 行保留作链路历史,复绑可 INSERT 新行。
-- 裁定: docs/notes/adopted/2026-08-20-db-dualtrack-convergence.md
BEGIN;

ALTER TABLE quad_links DROP CONSTRAINT quad_links_asset_id_key;
ALTER TABLE quad_links DROP CONSTRAINT quad_links_customer_id_key;
ALTER TABLE quad_links DROP CONSTRAINT quad_links_port_id_key;
ALTER TABLE quad_links DROP CONSTRAINT quad_links_address_id_key;

CREATE UNIQUE INDEX uq_quad_links_asset_active ON quad_links(asset_id) WHERE status IN ('LINKED','CONFLICT');
CREATE UNIQUE INDEX uq_quad_links_customer_active ON quad_links(customer_id) WHERE status IN ('LINKED','CONFLICT');
CREATE UNIQUE INDEX uq_quad_links_port_active ON quad_links(port_id) WHERE status IN ('LINKED','CONFLICT');
CREATE UNIQUE INDEX uq_quad_links_address_active ON quad_links(address_id) WHERE status IN ('LINKED','CONFLICT');

COMMIT;
