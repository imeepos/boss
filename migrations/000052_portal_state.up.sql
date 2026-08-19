-- 用户端/师傅端门户真实状态落库:验证码/门户账号/偏好/消息/钱包/单号序列。
-- 取代 handler 进程内 map(重启即失、多副本不可用);正式环境唯一事实源为 DB。
BEGIN;

-- 验证码:用户端(login/register/reset)与师傅端(login)共用;一次性消费。
CREATE TABLE portal_sms_codes (
    phone      VARCHAR(32)  NOT NULL,
    scene      VARCHAR(16)  NOT NULL,            -- login/register/reset
    code       VARCHAR(8)   NOT NULL,
    expires_at TIMESTAMPTZ  NOT NULL,
    used       BOOLEAN      NOT NULL DEFAULT FALSE,
    PRIMARY KEY (phone, scene)
);

-- 门户账号:手机号登录,与 customers 1:1(customer_id 可为隔离空间合成 ID)。
CREATE TABLE portal_accounts (
    phone                VARCHAR(32) PRIMARY KEY,
    customer_id          BIGINT      NOT NULL UNIQUE,
    password_hash        TEXT        NOT NULL,
    password_updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 通知偏好/语言。
CREATE TABLE portal_prefs (
    customer_id BIGINT PRIMARY KEY,
    notify      JSONB       NOT NULL DEFAULT '{}'::jsonb,
    language    VARCHAR(16) NOT NULL DEFAULT ''
);

-- 站内消息:payload 原样 JSONB(契约字段由端定义)。
CREATE TABLE portal_messages (
    id          BIGSERIAL PRIMARY KEY,
    customer_id BIGINT      NOT NULL,
    payload     JSONB       NOT NULL,
    read        BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_portal_messages_customer ON portal_messages(customer_id, read);

-- 门户钱包余额(充值/缴费);金额 numeric(12,2)。
CREATE TABLE portal_wallets (
    customer_id BIGINT PRIMARY KEY,
    balance     NUMERIC(12,2) NOT NULL DEFAULT 0
);

-- 业务单号序列:PAY(缴费)/CHG(催单)/TKT(报修)/MSG(消息)。
CREATE TABLE portal_seq (
    kind VARCHAR(8) PRIMARY KEY,
    last BIGINT NOT NULL DEFAULT 0
);
INSERT INTO portal_seq(kind) VALUES ('PAY'),('CHG'),('TKT'),('MSG'),('CUST');

COMMIT;
