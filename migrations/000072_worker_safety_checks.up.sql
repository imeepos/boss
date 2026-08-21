-- 师傅安全作业确认留痕(POST /safety/checks 此前仅审计日志,线上运营需可检索的结构化记录)。
BEGIN;

CREATE TABLE worker_safety_checks (
    id          BIGSERIAL PRIMARY KEY,
    worker_id   BIGINT       NOT NULL REFERENCES workers(id),
    work_type   VARCHAR(32)  NOT NULL,               -- 作业类型(登高/井下/带电等)
    checklist   TEXT[]       NOT NULL,               -- 勾选项编码
    checked_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_worker_safety_worker ON worker_safety_checks(worker_id, checked_at DESC);

COMMIT;
