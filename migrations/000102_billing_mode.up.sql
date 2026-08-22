-- 套餐付费模式(adopted note 2026-08-22-prepaid-postpaid-billing-mode):
-- 挂客户订购关系(lo_accounts)与订单快照(orders),不挂 product_offers;
-- 存量数据一律后付费。预付费出账排除(GenerateBills),环节4 当场收款。
BEGIN;

ALTER TABLE lo_accounts
    ADD COLUMN billing_mode VARCHAR(16) NOT NULL DEFAULT 'POSTPAID'
    CONSTRAINT ck_lo_accounts_billing_mode CHECK (billing_mode IN ('PREPAID', 'POSTPAID'));

ALTER TABLE orders
    ADD COLUMN billing_mode VARCHAR(16) NOT NULL DEFAULT 'POSTPAID'
    CONSTRAINT ck_orders_billing_mode CHECK (billing_mode IN ('PREPAID', 'POSTPAID'));

COMMIT;
