-- user_plans 增加合约到期月(契约 Home.plan.contractEnd / Plan.contractEnd,示例 '2026-08')。
-- 存量行按默认合约期 24 个月自生效月推算回填;新行缺省 NULL(无合约)。
ALTER TABLE user_plans ADD COLUMN contract_end VARCHAR(7);

UPDATE user_plans
SET contract_end = to_char(effective_at + interval '24 months', 'YYYY-MM')
WHERE contract_end IS NULL;
