-- 000187 down:字典为增量能力,回滚删表与引用列(存量 type 冗余列不受影响)。
BEGIN;
ALTER TABLE assets DROP CONSTRAINT IF EXISTS assets_model_id_fkey;
ALTER TABLE assets DROP COLUMN IF EXISTS model_id;
DROP TABLE IF EXISTS asset_models;
COMMIT;
