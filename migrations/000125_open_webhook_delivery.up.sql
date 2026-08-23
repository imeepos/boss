-- 开放平台 M2:Webhook 投递 outbox(docs/plan/q4-open-platform-plan.md)。
-- 投递语义:
--   幂等:同订阅同事件(event_id)只投一次,UNIQUE 约束 + ON CONFLICT DO NOTHING;
--   重试:指数退避( attempts 越多重试越晚),超过上限进死信(status=2)人工介入;
--   签名:负载经应用 Secret HMAC 签名(SignPayload),接收方可验真。

CREATE TABLE open_webhook_deliveries (
    id              BIGSERIAL PRIMARY KEY,
    subscription_id BIGINT NOT NULL REFERENCES open_webhook_subscriptions(id) ON DELETE CASCADE,
    event_id        TEXT NOT NULL,                    -- 幂等键,如 ORD-20250817-001:activated
    event_type      TEXT NOT NULL,                    -- 如 order.activated
    payload         JSONB NOT NULL,
    status          SMALLINT NOT NULL DEFAULT 0,      -- 0待投递 1已投递 2死信(超最大重试)
    attempts        INT NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    http_status     INT,
    last_error      TEXT,
    delivered_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (subscription_id, event_id)
);

CREATE INDEX idx_open_webhook_deliveries_due
    ON open_webhook_deliveries (status, next_attempt_at);
