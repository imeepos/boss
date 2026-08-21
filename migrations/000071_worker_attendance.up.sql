-- 师傅考勤打卡流水(师傅端 POST /schedule/clock 此前只回执不落库,线上运营需留痕)。
-- clock_type 对齐契约 api/openapi/worker/profile.yaml 枚举 IN/OUT。
BEGIN;

CREATE TABLE worker_attendance (
    id         BIGSERIAL PRIMARY KEY,
    worker_id  BIGINT      NOT NULL REFERENCES workers(id),
    clock_type VARCHAR(8)  NOT NULL,               -- IN 上班 / OUT 下班
    clocked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_worker_attendance_type CHECK (clock_type IN ('IN', 'OUT'))
);
CREATE INDEX idx_worker_attendance_worker ON worker_attendance(worker_id, clocked_at DESC);

COMMIT;
