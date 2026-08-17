-- 阶段5:应收信用(AR)+计费(BIL)。欠费快照 + 停复机流水。
-- 字段权威:server-ts/src/entities/worker.ts(Arrears/StopResumeTask)。
BEGIN;

CREATE TABLE arrears (
    id          BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL UNIQUE REFERENCES customers(id),
    amount      NUMERIC(12,2) NOT NULL,
    days        INTEGER NOT NULL,
    status      VARCHAR(16) NOT NULL  -- 催收中/已停机等(应收信用域)
);

CREATE TABLE stop_resume_tasks (
    id           BIGSERIAL PRIMARY KEY,
    customer_id  BIGINT NOT NULL,          -- 软引用 customers.id
    lo_account_id BIGINT NOT NULL,         -- 软引用 lo_accounts.id
    action       VARCHAR(8) NOT NULL,      -- STOP停机/RESUME复机
    status       VARCHAR(16) NOT NULL      -- PENDING/DOING/DONE/FAILED
);
CREATE INDEX idx_stop_resume_customer ON stop_resume_tasks(customer_id);

COMMIT;
