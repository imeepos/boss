-- D2(db-design-review): portal_accounts.customer_id 号段裁定落地。
-- 正数=真实 customers.id;负数=隔离空间合成 ID;0 非法。CHECK 强制防撞号。
-- 裁定: docs/notes/adopted/2026-08-20-db-dualtrack-convergence.md
BEGIN;

ALTER TABLE portal_accounts ADD CONSTRAINT chk_portal_customer_id_sign
    CHECK (customer_id <> 0);

COMMIT;
