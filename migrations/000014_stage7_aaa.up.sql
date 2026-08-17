-- 阶段7:AAA 认证计费(LO 账号 / 话单 CDR / 认证日志)。
-- 字段权威:server-ts/src/entities/oss.ts(LoAccount) + aaa.ts(CallDetailRecord/AuthLog)。
BEGIN;

CREATE TABLE lo_accounts (
    id                BIGSERIAL PRIMARY KEY,
    loid              VARCHAR(32) NOT NULL UNIQUE,  -- LOID-88A1
    customer_id       BIGINT NOT NULL,              -- → customers(1:1)
    legal_entity_id   BIGINT NOT NULL,
    legal_entity_name VARCHAR(128) NOT NULL,
    region_id         INTEGER NOT NULL,
    region_name       VARCHAR(64) NOT NULL,
    offer_id          BIGINT NOT NULL,              -- → product_offers
    qos_template_id   BIGINT NOT NULL,              -- 软引用 qos_templates
    status            VARCHAR(16) NOT NULL DEFAULT 'ACTIVE' -- ACTIVE/SUSPENDED/CLOSED
);
CREATE INDEX idx_lo_accounts_customer ON lo_accounts(customer_id);

CREATE TABLE cdrs (
    id             BIGSERIAL PRIMARY KEY,
    loid           VARCHAR(32) NOT NULL,            -- 认证账号 LOID
    username       VARCHAR(64),
    acct_status    SMALLINT NOT NULL,               -- 1开始/2停止/3中间
    session_id     VARCHAR(64),
    session_time   INTEGER NOT NULL,                -- 秒
    input_octets   BIGINT NOT NULL,
    output_octets  BIGINT NOT NULL,
    nas_ip         VARCHAR(64),
    billing_status VARCHAR(16) NOT NULL DEFAULT 'UNBILLED', -- UNBILLED/BILLED
    started_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_cdrs_loid ON cdrs(loid, started_at);

CREATE TABLE auth_logs (
    id         BIGSERIAL PRIMARY KEY,
    loid       VARCHAR(32) NOT NULL,
    result     VARCHAR(16) NOT NULL,                -- SUCCESS/FAILED
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_auth_logs_loid ON auth_logs(loid, created_at);

COMMIT;
