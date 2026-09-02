-- 回滚仅摘约束;已归一的大写值不回写小写——应用层自 000177 起只写大写,
-- 回写小写会立刻再度违反唯一口径(2026-09 schema 审计)。
BEGIN;

ALTER TABLE coupons DROP CONSTRAINT IF EXISTS ck_coupons_status;

COMMIT;
