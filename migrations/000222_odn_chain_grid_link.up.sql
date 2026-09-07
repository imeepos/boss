BEGIN;
ALTER TABLE odn_resource_chain
    ADD COLUMN prv_code CHAR(6),
    ADD COLUMN city_prefix VARCHAR(5),
    ADD COLUMN grid_code SMALLINT,
    ADD CONSTRAINT chk_odn_chain_attribution_all_or_none
        CHECK ((prv_code IS NULL AND city_prefix IS NULL AND grid_code IS NULL)
            OR (prv_code IS NOT NULL AND city_prefix IS NOT NULL AND grid_code IS NOT NULL)),
    ADD CONSTRAINT chk_odn_chain_grid_range CHECK (grid_code IS NULL OR grid_code BETWEEN 1 AND 99),
    ADD CONSTRAINT fk_odn_chain_grid
        FOREIGN KEY (prv_code, city_prefix, grid_code)
        REFERENCES odn_grid (prv_code, city_prefix, grid_code);
CREATE INDEX idx_odn_chain_attribution ON odn_resource_chain (prv_code, city_prefix, grid_code)
    WHERE grid_code IS NOT NULL;
COMMIT;
