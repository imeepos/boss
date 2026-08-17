-- 阶段7:配置下发(下发模板 / 下发任务 / 下发日志)。
-- 字段权威:server-ts/src/entities/oss.ts(ProvisionTemplate/ProvisionTask/ProvisionLog)。
BEGIN;

CREATE TABLE provision_templates (
    id              BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities(id),
    code            VARCHAR(32) NOT NULL UNIQUE,  -- TPL-FTTH
    name            VARCHAR(64) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE provision_tasks (
    id            BIGSERIAL PRIMARY KEY,
    lo_account_id BIGINT NOT NULL,          -- 软引用 lo_accounts
    template_id   BIGINT NOT NULL REFERENCES provision_templates(id),
    status        VARCHAR(16) NOT NULL DEFAULT 'PENDING', -- PENDING/DOING/DONE/FAILED
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_provision_tasks_template ON provision_tasks(template_id);

CREATE TABLE provision_logs (
    id            BIGSERIAL PRIMARY KEY,
    task_id       BIGINT NOT NULL,          -- 软引用 provision_tasks
    resource_id   BIGINT NOT NULL,          -- 软引用 resources
    resource_code VARCHAR(64),
    template_id   BIGINT NOT NULL,          -- 软引用 provision_templates
    template_code VARCHAR(64),
    result        VARCHAR(16) NOT NULL,     -- SUCCESS/FAILED
    retries       SMALLINT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_provision_logs_task ON provision_logs(task_id, created_at);

COMMIT;
