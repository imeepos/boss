ALTER TABLE orders
    DROP COLUMN IF EXISTS buy_months,
    DROP COLUMN IF EXISTS gift_months;
