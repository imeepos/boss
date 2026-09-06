-- 回滚 ODN 生命周期列。
BEGIN;

ALTER TABLE odn_device  DROP COLUMN IF EXISTS lifecycle_status;
ALTER TABLE odn_site    DROP COLUMN IF EXISTS lifecycle_status;
ALTER TABLE odn_facility DROP COLUMN IF EXISTS lifecycle_status;

COMMIT;
