-- 后台录入企业员工登录信息:accounts 增加工号(staff_no),对标 workers.staff_no(000016)。
-- 工号可空(存量后台账号/入驻开通的 partner_admin 无工号),非空时全局唯一(登录标识语义)。
-- 字段权威:docs/contract/fields.md 1.2(本迁移同步登记)。
BEGIN;

ALTER TABLE accounts ADD COLUMN staff_no VARCHAR(32);
CREATE UNIQUE INDEX uq_accounts_staff_no ON accounts(staff_no) WHERE staff_no IS NOT NULL;

COMMIT;
