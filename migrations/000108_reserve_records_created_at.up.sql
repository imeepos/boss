-- Q2 订单预占超时释放:reserve_records 补预占发生时间,供超时判定。
-- 参数默认值见 docs/plan/3-month-roadmap.md §2(端口预占有效期 30 分钟,可经 biz_params 调)。
BEGIN;

ALTER TABLE reserve_records
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX idx_reserve_status_created ON reserve_records(status, created_at);

INSERT INTO biz_params(key, value, description) VALUES
    ('order.reserve.timeoutMinutes', '30', '端口预占有效期(分钟):RESERVED 订单超时自动释放端口并回 PENDING')
ON CONFLICT (key) DO NOTHING;

COMMIT;
