-- Q1 CS/AR 基础：客服工单闭环、应收信用策略与可回放指标事实。
-- 域边界：CS 负责服务工单及服务体验；AR 负责账龄、信用和催收动作；跨域仅以 customer/order/bill/dispatch_ticket 引用。
BEGIN;

CREATE TABLE cs_ticket_extensions (
    ticket_id BIGINT PRIMARY KEY REFERENCES complaints(id),
    priority VARCHAR(16) NOT NULL DEFAULT 'NORMAL',
    escalation_level SMALLINT NOT NULL DEFAULT 0,
    first_response_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ,
    sla_due_at TIMESTAMPTZ,
    resolution_code VARCHAR(32),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_cs_ticket_sla ON cs_ticket_extensions(sla_due_at) WHERE resolved_at IS NULL;

CREATE TABLE cs_ticket_events (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT NOT NULL REFERENCES complaints(id),
    event_type VARCHAR(32) NOT NULL,
    from_status VARCHAR(16),
    to_status VARCHAR(16),
    actor_id BIGINT,
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_cs_ticket_events_ticket ON cs_ticket_events(ticket_id, created_at);

CREATE TABLE cs_knowledge_articles (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(32) NOT NULL UNIQUE,
    title VARCHAR(160) NOT NULL,
    content TEXT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'DRAFT',
    version INTEGER NOT NULL DEFAULT 1,
    owner_id BIGINT,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE cs_callbacks (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT NOT NULL REFERENCES complaints(id),
    customer_id BIGINT NOT NULL REFERENCES customers(id),
    scheduled_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    result VARCHAR(16),
    rating SMALLINT,
    comment TEXT,
    operator_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_cs_callbacks_pending ON cs_callbacks(scheduled_at) WHERE completed_at IS NULL;

CREATE TABLE ar_aging_snapshots (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers(id),
    snapshot_date DATE NOT NULL,
    current_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    days_1_30 NUMERIC(12,2) NOT NULL DEFAULT 0,
    days_31_60 NUMERIC(12,2) NOT NULL DEFAULT 0,
    days_61_90 NUMERIC(12,2) NOT NULL DEFAULT 0,
    days_90_plus NUMERIC(12,2) NOT NULL DEFAULT 0,
    total_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(customer_id, snapshot_date)
);
CREATE INDEX idx_ar_aging_date ON ar_aging_snapshots(snapshot_date, customer_id);

CREATE TABLE ar_credit_profiles (
    customer_id BIGINT PRIMARY KEY REFERENCES customers(id),
    credit_level VARCHAR(16) NOT NULL DEFAULT 'STANDARD',
    risk_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    stop_threshold NUMERIC(12,2) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ar_collection_tasks (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers(id),
    aging_snapshot_id BIGINT REFERENCES ar_aging_snapshots(id),
    task_type VARCHAR(16) NOT NULL,
    priority VARCHAR(16) NOT NULL DEFAULT 'NORMAL',
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    due_at TIMESTAMPTZ NOT NULL,
    assigned_to BIGINT,
    completed_at TIMESTAMPTZ,
    outcome VARCHAR(32),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_ar_collection_queue ON ar_collection_tasks(status, priority, due_at);

CREATE TABLE ar_payment_promises (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers(id),
    amount NUMERIC(12,2) NOT NULL,
    promised_at TIMESTAMPTZ NOT NULL,
    due_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'OPEN',
    fulfilled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_ar_promises_due ON ar_payment_promises(status, due_at);

CREATE TABLE ar_writeoffs (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers(id),
    amount NUMERIC(12,2) NOT NULL,
    reason VARCHAR(32) NOT NULL,
    approved_by BIGINT,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE service_metric_snapshots (
    id BIGSERIAL PRIMARY KEY,
    metric_date DATE NOT NULL,
    metric_key VARCHAR(48) NOT NULL,
    numerator NUMERIC(18,4) NOT NULL DEFAULT 0,
    denominator NUMERIC(18,4) NOT NULL DEFAULT 0,
    value NUMERIC(18,4) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(metric_date, metric_key)
);

COMMIT;
