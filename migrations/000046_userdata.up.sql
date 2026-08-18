-- 用户端数据(userdata)域:承接 api/openapi/admin/userdata.yaml。
-- 用户主档为 customers(000010 已建);本迁移建用户端专属子表与全局配置表。
-- 金额一律 BIGINT 分;外键挂 customers.id。
BEGIN;

CREATE TABLE user_accounts (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    login_name VARCHAR(64) NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    password_updated_at TIMESTAMPTZ,
    auto_pay BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (login_name)
);

CREATE TABLE user_addresses (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    addr_code VARCHAR(32) NOT NULL,
    contact VARCHAR(64) NOT NULL,
    phone VARCHAR(32) NOT NULL,
    detail VARCHAR(255) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE user_plans (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    product_id BIGINT NOT NULL,
    plan_name VARCHAR(128) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    effective_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE addons (
    addon_id VARCHAR(32) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    price BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(8) NOT NULL DEFAULT 'on' CHECK (status IN ('on', 'off')),
    subscriber_count BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE addon_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    addon_id VARCHAR(32) NOT NULL REFERENCES addons (addon_id),
    action VARCHAR(16) NOT NULL DEFAULT 'subscribe' CHECK (action IN ('subscribe', 'unsubscribe')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_notify_settings (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL UNIQUE REFERENCES customers (id),
    business BOOLEAN NOT NULL DEFAULT TRUE,
    marketing BOOLEAN NOT NULL DEFAULT FALSE,
    channel VARCHAR(16) NOT NULL DEFAULT 'app' CHECK (channel IN ('sms', 'app', 'both'))
);

CREATE TABLE user_faqs (
    faq_id VARCHAR(32) PRIMARY KEY,
    category VARCHAR(64) NOT NULL,
    question VARCHAR(255) NOT NULL,
    answer TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE user_messages (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    type VARCHAR(16) NOT NULL DEFAULT 'notice',
    title VARCHAR(128) NOT NULL,
    content TEXT NOT NULL,
    read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE coupons (
    coupon_id VARCHAR(32) PRIMARY KEY,
    customer_id BIGINT REFERENCES customers (id),
    name VARCHAR(128) NOT NULL,
    amount BIGINT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    expire_at TIMESTAMPTZ
);

CREATE TABLE invite_config (
    id BIGSERIAL PRIMARY KEY,
    invite_link VARCHAR(255) NOT NULL,
    reward_amount BIGINT NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT TRUE
);
INSERT INTO invite_config (invite_link, reward_amount) VALUES ('https://u.ymm.example/invite', 1000);

CREATE TABLE user_usages (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    month VARCHAR(7) NOT NULL,
    upload_gb BIGINT NOT NULL DEFAULT 0,
    download_gb BIGINT NOT NULL DEFAULT 0,
    total_gb BIGINT NOT NULL DEFAULT 0,
    UNIQUE (customer_id, month)
);

CREATE TABLE diy_guides (
    guide_id VARCHAR(32) PRIMARY KEY,
    title VARCHAR(128) NOT NULL,
    category VARCHAR(64) NOT NULL,
    steps TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE agreements (
    agreement_id VARCHAR(32) PRIMARY KEY,
    type VARCHAR(32) NOT NULL,
    version VARCHAR(16) NOT NULL,
    content TEXT NOT NULL,
    effective_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_balances (
    customer_id BIGINT PRIMARY KEY REFERENCES customers (id),
    balance BIGINT NOT NULL DEFAULT 0,
    warn_line BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE topup_denominations (
    denom_id VARCHAR(32) PRIMARY KEY,
    amount BIGINT NOT NULL,
    bonus BIGINT NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE user_invoices (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    bill_no VARCHAR(32) NOT NULL,
    invoice_no VARCHAR(32) NOT NULL,
    amount BIGINT NOT NULL,
    title VARCHAR(128) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'ISSUED'
);

CREATE TABLE user_complaints (
    complaint_id VARCHAR(32) PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    type VARCHAR(32) NOT NULL,
    content TEXT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'OPEN'
);

CREATE TABLE user_verify_records (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    step VARCHAR(32) NOT NULL,
    result VARCHAR(16) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE product_specs (
    product_id VARCHAR(32) PRIMARY KEY,
    highlights TEXT NOT NULL,
    specs TEXT NOT NULL
);

CREATE TABLE user_bill_items (
    id BIGSERIAL PRIMARY KEY,
    bill_no VARCHAR(32) NOT NULL,
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    name VARCHAR(128) NOT NULL,
    amount BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_user_accounts_customer ON user_accounts (customer_id);
CREATE INDEX idx_user_addresses_customer ON user_addresses (customer_id);
CREATE INDEX idx_user_plans_customer ON user_plans (customer_id);
CREATE INDEX idx_user_messages_customer ON user_messages (customer_id);
CREATE INDEX idx_user_bill_items_bill_no ON user_bill_items (bill_no);
CREATE INDEX idx_user_invoices_bill_no ON user_invoices (bill_no);

COMMIT;
