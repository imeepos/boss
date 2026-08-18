-- 阶段9:自动报告快照(操作留痕;报告正文为派生聚合,同窗口幂等覆盖)。
BEGIN;

CREATE TABLE report_snapshots (
    id           BIGSERIAL PRIMARY KEY,
    period       VARCHAR(16) NOT NULL,     -- daily/weekly/monthly/quarterly
    window_start TIMESTAMPTZ NOT NULL,
    window_end   TIMESTAMPTZ NOT NULL,
    payload      JSONB NOT NULL,           -- 指标/ROI/热力/维护/结论 全文
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_report_period_window UNIQUE (period, window_start)
);

COMMIT;
