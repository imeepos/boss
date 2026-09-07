-- 000204 down: 移除结算单与工程量清单列。
BEGIN;

DROP INDEX IF EXISTS uq_construction_settlements_project_active;
DROP INDEX IF EXISTS idx_construction_settlements_status;
DROP INDEX IF EXISTS idx_construction_settlements_project;
DROP TABLE IF EXISTS construction_settlements;

ALTER TABLE construction_items
    DROP COLUMN IF EXISTS amount,
    DROP COLUMN IF EXISTS unit_price,
    DROP COLUMN IF EXISTS quantity;

COMMIT;
