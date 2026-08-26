-- 双挂兜底:账单缴费落账行若缺 customer_id,按 bill 回填(000068 已回填当时存量,
-- 后由未带 customer_id 落账路径新增的空行再补);源头已由 RecordPaymentWithCoupon 强制双挂。
UPDATE payments p SET customer_id = b.customer_id
  FROM bills b
 WHERE p.bill_id = b.id AND p.customer_id IS NULL;
