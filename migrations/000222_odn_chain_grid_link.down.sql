BEGIN;
ALTER TABLE odn_resource_chain
    DROP CONSTRAINT IF EXISTS fk_odn_chain_grid,
    DROP CONSTRAINT IF EXISTS chk_odn_chain_grid_range,
    DROP CONSTRAINT IF EXISTS chk_odn_chain_attribution_all_or_none,
    DROP COLUMN IF EXISTS prv_code,
    DROP COLUMN IF EXISTS city_prefix,
    DROP COLUMN IF EXISTS grid_code;
COMMIT;
