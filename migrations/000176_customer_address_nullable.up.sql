-- 000176: customers.address_id 放开可空——客户可先建档后补地址。
-- 死循环修复(2026-09-01):POST /customers 原强制 addressId,而内联建址/用户地址
-- 均要求 customerId 先存在,新建客户与建址互为前置,受理台无从起步。
-- 建档时地址可缺省(0/空),维护地址走 POST /orders/address backfillCustomer=true 回填。
ALTER TABLE customers ALTER COLUMN address_id DROP NOT NULL;
