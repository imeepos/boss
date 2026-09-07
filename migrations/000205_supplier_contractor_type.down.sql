-- 000203 down: 移除供应商承建类型维度。
ALTER TABLE procurement_suppliers
    DROP COLUMN IF EXISTS qualification,
    DROP COLUMN IF EXISTS contractor_type;
