-- 000219 down: 移除工程应付台账四表。
BEGIN;

DROP TABLE IF EXISTS construction_payable_invoices;
DROP TABLE IF EXISTS construction_payable_deductions;
DROP TABLE IF EXISTS construction_payable_payments;
DROP TABLE IF EXISTS construction_payables;

COMMIT;
