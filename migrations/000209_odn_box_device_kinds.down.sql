-- 回滚字典扩展:先清导入域设备(无城市行),再还原原约束与 NOT NULL。
-- 注意:若导入域设备被外部表(如 address_coverage)引用,须先清理引用行。
BEGIN;
DROP INDEX IF EXISTS uq_odn_device_box;
DELETE FROM odn_device WHERE prv_code IS NULL;
ALTER TABLE odn_device DROP CONSTRAINT IF EXISTS odn_device_city_required_check;
ALTER TABLE odn_device DROP CONSTRAINT IF EXISTS odn_device_code_check;
ALTER TABLE odn_device DROP CONSTRAINT IF EXISTS odn_device_kind_check;
ALTER TABLE odn_device ADD CONSTRAINT odn_device_code_check
    CHECK (code ~ '^(SNW|OLT|ODF|OCC|ODB|SDB|PRT|TBP)[0-9]{3}(-([2-9]|[1-9][0-9]+))?$');
ALTER TABLE odn_device ADD CONSTRAINT odn_device_kind_check
    CHECK (kind IN ('SNW','OLT','ODF','OCC','ODB','SDB','PRT','TBP'));
ALTER TABLE odn_device ALTER COLUMN prv_code SET NOT NULL;
ALTER TABLE odn_device ALTER COLUMN city_prefix SET NOT NULL;
COMMIT;
