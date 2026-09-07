-- 000216 down:恢复四态约束;IN_TRANSIT 在途回 IN_STOCK(降级路径,业务侧已无出库单)。

BEGIN;

UPDATE assets SET status = 'IN_STOCK' WHERE status = 'IN_TRANSIT';

ALTER TABLE assets DROP CONSTRAINT ck_assets_status;
ALTER TABLE assets ADD CONSTRAINT ck_assets_status
  CHECK (status IN ('IN_STOCK','DEPLOYED','MAINTENANCE','SCRAPPED'));

COMMIT;
