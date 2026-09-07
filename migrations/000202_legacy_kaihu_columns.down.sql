-- 迁移 000202 回滚:逆序撤销(先 lo_accounts 后 ports,列序与 up 完全对称)。
BEGIN;

ALTER TABLE lo_accounts DROP COLUMN IF EXISTS contract_months;

ALTER TABLE ports DROP COLUMN IF EXISTS legacy_path;
ALTER TABLE ports DROP COLUMN IF EXISTS tr069_cvlan;
ALTER TABLE ports DROP COLUMN IF EXISTS internet_cvlan;
ALTER TABLE ports DROP COLUMN IF EXISTS cvlan;
ALTER TABLE ports DROP COLUMN IF EXISTS svlan;

COMMIT;
