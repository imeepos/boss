-- 门户缴费偏好:自动缴费开通状态按客户落库(重启/多副本不丢)。
BEGIN;

CREATE TABLE portal_billing_prefs (
    customer_id      BIGINT PRIMARY KEY,
    auto_pay_enabled BOOLEAN     NOT NULL DEFAULT FALSE,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;
