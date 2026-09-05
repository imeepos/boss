-- 000184 down:仅摘除闸门约束;已清洗的合法值不回滚为脏值。
BEGIN;
ALTER TABLE assets DROP CONSTRAINT IF EXISTS ck_assets_status;
ALTER TABLE assets DROP CONSTRAINT IF EXISTS ck_assets_type_notblank;
ALTER TABLE tags DROP CONSTRAINT IF EXISTS ck_tags_status;
COMMIT;
