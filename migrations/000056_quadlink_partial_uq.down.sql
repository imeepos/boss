BEGIN;

DROP INDEX uq_quad_links_asset_active;
DROP INDEX uq_quad_links_customer_active;
DROP INDEX uq_quad_links_port_active;
DROP INDEX uq_quad_links_address_active;

ALTER TABLE quad_links ADD CONSTRAINT quad_links_asset_id_key UNIQUE (asset_id);
ALTER TABLE quad_links ADD CONSTRAINT quad_links_customer_id_key UNIQUE (customer_id);
ALTER TABLE quad_links ADD CONSTRAINT quad_links_port_id_key UNIQUE (port_id);
ALTER TABLE quad_links ADD CONSTRAINT quad_links_address_id_key UNIQUE (address_id);

COMMIT;
