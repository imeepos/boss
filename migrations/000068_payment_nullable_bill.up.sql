-- 充值流水:topup 无关联账单,bill_id 允许 NULL(缴费流水仍指向 bills.id);
-- 并补 customer_id 使无账单流水的归属可查(存量按 bills 回填)。
ALTER TABLE payments ALTER COLUMN bill_id DROP NOT NULL;
ALTER TABLE payments ADD COLUMN customer_id BIGINT REFERENCES customers(id);
UPDATE payments p SET customer_id = b.customer_id
  FROM bills b WHERE p.bill_id = b.id AND p.customer_id IS NULL;
CREATE INDEX idx_payments_customer ON payments(customer_id);
