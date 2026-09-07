-- 000223: 勘测任务域(P-INFRA-1 W7,F5b):admin 创建/指派勘测任务,师傅端可见可接,
-- 现场回填(打点坐标/照片/设施状态备注/建议)append-only 留痕,作规划/备案输入。
BEGIN;
CREATE TABLE survey_tasks (
    id                 BIGSERIAL PRIMARY KEY,
    task_no            VARCHAR(32) NOT NULL UNIQUE,
    title              VARCHAR(128) NOT NULL,
    description        VARCHAR(500) NOT NULL DEFAULT '',
    prv_code           VARCHAR(8) NOT NULL DEFAULT '',
    city_prefix        VARCHAR(8) NOT NULL DEFAULT '',
    grid_code          INTEGER NOT NULL DEFAULT 0,
    assigned_worker_id BIGINT REFERENCES workers(id),
    status             VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    created_by         BIGINT REFERENCES accounts,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_survey_tasks_worker ON survey_tasks (assigned_worker_id, status);
CREATE TABLE survey_task_reports (
    id             BIGSERIAL PRIMARY KEY,
    task_id        BIGINT NOT NULL REFERENCES survey_tasks(id),
    worker_id      BIGINT NOT NULL REFERENCES workers(id),
    lat            DOUBLE PRECISION,
    lng            DOUBLE PRECISION,
    facility_note  VARCHAR(500) NOT NULL DEFAULT '',
    suggestion     VARCHAR(24) NOT NULL CHECK (suggestion IN ('CAN_INSTALL','NEED_NEW_FACILITY')),
    photo_ids      BIGINT[] NOT NULL DEFAULT '{}',
    client_msg_id  VARCHAR(64) NOT NULL,
    reported_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (task_id, client_msg_id)
);
CREATE INDEX idx_survey_reports_task ON survey_task_reports (task_id, id DESC);
COMMIT;