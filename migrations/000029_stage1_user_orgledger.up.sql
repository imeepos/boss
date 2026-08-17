-- 阶段1:账号组织归属台账(调岗/调部门/调公司的时间段)。
-- 字段权威:server-ts/src/entities/org.ts(AccountOrgHistory)。
BEGIN;

CREATE TABLE account_org_histories (
    id                 BIGSERIAL PRIMARY KEY,
    account_id         BIGINT NOT NULL REFERENCES accounts(id),
    legal_entity_id    BIGINT REFERENCES legal_entities(id),
    legal_entity_name  VARCHAR(128),
    dept_id            BIGINT REFERENCES departments(id),
    dept_name          VARCHAR(64),
    post_id            BIGINT REFERENCES posts(id),
    post_name          VARCHAR(64),
    reason             VARCHAR(128),
    operator_account_id BIGINT,
    effective_from     TIMESTAMPTZ NOT NULL,
    effective_to       TIMESTAMPTZ               -- null=至今
);
CREATE INDEX idx_account_org_histories ON account_org_histories(account_id, effective_from);

COMMIT;
