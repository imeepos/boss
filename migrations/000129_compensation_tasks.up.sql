-- S2 统一补偿任务/责任队列:全量失败来源汇聚,支持领取/转派/重试/回放/关闭/审计。
BEGIN;
CREATE TABLE IF NOT EXISTS compensation_tasks (
    id              BIGSERIAL PRIMARY KEY,
    source          VARCHAR(32) NOT NULL DEFAULT '',
    biz_type        VARCHAR(32) NOT NULL,
    biz_id          VARCHAR(64) NOT NULL,
    failure_reason  TEXT NOT NULL DEFAULT '',
    priority        VARCHAR(8) NOT NULL DEFAULT 'NORMAL',
    status          VARCHAR(16) NOT NULL DEFAULT 'OPEN',
    assignee_id     BIGINT,
    assignee_name   VARCHAR(64),
    sla_deadline    TIMESTAMPTZ,
    claimed_at      TIMESTAMPTZ,
    claimed_by      BIGINT,
    closed_at       TIMESTAMPTZ,
    closed_by       BIGINT,
    close_reason    TEXT,
    retry_count     INT NOT NULL DEFAULT 0,
    max_retries     INT NOT NULL DEFAULT 3,
    last_retry_at   TIMESTAMPTZ,
    audit_log       JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_comp_tasks_status ON compensation_tasks(status);
CREATE INDEX IF NOT EXISTS idx_comp_tasks_source ON compensation_tasks(source);
CREATE INDEX IF NOT EXISTS idx_comp_tasks_assignee ON compensation_tasks(assignee_id) WHERE assignee_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_comp_tasks_sla ON compensation_tasks(sla_deadline) WHERE sla_deadline IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_comp_tasks_biz ON compensation_tasks(biz_type, biz_id);

COMMENT ON TABLE compensation_tasks IS 'S2 统一补偿任务/责任队列:失败来源汇聚 + 人工/自动补偿工作流';
COMMENT ON COLUMN compensation_tasks.source IS '失败来源:order/payment/cdr/tax/quadlink/provision/points/coupon/webhook/patrol';
COMMENT ON COLUMN compensation_tasks.biz_type IS '业务对象类型:order/payment/cdr/invoice/quadlink/provision/points/coupon/webhook';
COMMENT ON COLUMN compensation_tasks.biz_id IS '业务对象ID:order_no/payment_no/cdr_id/invoice_no/quadlink_id/task_no/points_id/coupon_no/webhook_id';
COMMENT ON COLUMN compensation_tasks.priority IS 'LOW/NORMAL/HIGH/URGENT';
COMMENT ON COLUMN compensation_tasks.status IS 'OPEN/CLAIMED/DOING/DONE/CLOSED';
COMMENT ON COLUMN compensation_tasks.assignee_id IS '责任人 account_id';
COMMENT ON COLUMN compensation_tasks.audit_log IS '审计事件JSON数组:[{at,actor,action,detail}]';
COMMIT;