-- 告警批量复测任务(alarm.yaml POST /alarms/batch-retest,台风应急按片区下发)。
BEGIN;

CREATE TABLE alarm_retest_tasks (
    id         BIGSERIAL PRIMARY KEY,
    task_no    VARCHAR(32) NOT NULL,        -- RT-YYYYMMDD-NNNN
    scope      VARCHAR(64) NOT NULL,        -- 下发范围(片区/区域)
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;
