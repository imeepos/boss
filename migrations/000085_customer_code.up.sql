-- 阶段:契约对齐;为 customers 加 customer_code(展示冗余,与 customer_id 一一对应)。
-- 字段权威:docs/contract/fields.md §2.1(2026-08-21 adopted);决策 note:
--   docs/notes/adopted/2026-08-21-customer-code-in-quadlink.md
-- 生成规则:'C-' + id 左零 8 位(如 id=46 -> 'C-00000046');
-- 命名空间与 asset_code/port_code 同构;对账/扫码/外键仍以 id 为权威,code 仅展示。
-- 兜底:BEFORE INSERT 触发器基于 NEW.id 派生 code,应用层无需关心(POSTGRES BIGSERIAL
--   默认表达式在 BEFORE 触发器之前求值,NEW.id 已被 sequence 赋值)。
BEGIN;

ALTER TABLE customers
    ADD COLUMN customer_code VARCHAR(32);

UPDATE customers
SET customer_code = 'C-' || lpad(id::text, 8, '0')
WHERE customer_code IS NULL;

CREATE OR REPLACE FUNCTION customers_set_code() RETURNS trigger AS $$
BEGIN
    IF NEW.customer_code IS NULL OR NEW.customer_code = '' THEN
        NEW.customer_code := 'C-' || lpad(NEW.id::text, 8, '0');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_customers_set_code ON customers;
CREATE TRIGGER trg_customers_set_code
    BEFORE INSERT ON customers
    FOR EACH ROW EXECUTE FUNCTION customers_set_code();

ALTER TABLE customers
    ALTER COLUMN customer_code SET NOT NULL;

CREATE UNIQUE INDEX uq_customers_customer_code
    ON customers (customer_code);

COMMIT;