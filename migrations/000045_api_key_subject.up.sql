-- API key 主体扩展:支持三类身份(accounts/customers/workers)各自签发密钥。
-- 复合业务测试场景:customer 下单 → account 接待/审核/派单 → worker 上门扫码 → account 管理订单,
-- 每种身份一个 key,免登录切换主体。
-- 约定:
--   subject_type: account | worker | customer(与 000043 三表登录边界对齐)
--   subject_ref : 对应主体表主键(accounts.id / workers.id / customers.id)
BEGIN;

ALTER TABLE api_keys
    ADD COLUMN subject_type VARCHAR(16) NOT NULL DEFAULT 'account'
        CHECK (subject_type IN ('account', 'worker', 'customer')),
    ADD COLUMN subject_ref BIGINT NOT NULL DEFAULT 0;

-- 存量数据回填:旧 account_id 迁移到 subject_ref
UPDATE api_keys SET subject_ref = account_id WHERE subject_type = 'account' AND subject_ref = 0;

ALTER TABLE api_keys DROP COLUMN account_id;

CREATE INDEX idx_api_keys_subject ON api_keys (subject_type, subject_ref, status);

COMMENT ON COLUMN api_keys.subject_type IS '密钥主体类型:account=管理后台账号 worker=师傅 customer=客户';
COMMENT ON COLUMN api_keys.subject_ref IS '主体表主键;subject_type 决定指向 accounts/workers/customers';

COMMIT;