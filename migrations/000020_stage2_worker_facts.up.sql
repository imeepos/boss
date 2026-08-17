-- 阶段2:师傅月度事实表(绩效/佣金/考勤,粒度=师傅×月×班组×区域)。
-- 字段权威:docs/contract/fields.md §7.3 + server-ts/src/entities/worker.ts。
BEGIN;

CREATE TABLE worker_performances (
    id                BIGSERIAL PRIMARY KEY,
    worker_id         BIGINT NOT NULL REFERENCES workers(id),
    group_id          BIGINT NOT NULL REFERENCES worker_groups(id),
    group_name        VARCHAR(64) NOT NULL,
    legal_entity_id   BIGINT NOT NULL,
    legal_entity_name VARCHAR(128) NOT NULL,
    region_id         INTEGER NOT NULL,
    region_name       VARCHAR(64) NOT NULL,
    period            VARCHAR(16) NOT NULL,   -- 2026-08
    finished          INTEGER NOT NULL,
    on_time_rate      SMALLINT NOT NULL,
    score             NUMERIC(3,1) NOT NULL,
    CONSTRAINT uq_worker_perf UNIQUE (worker_id, period, group_id, region_id)
);
CREATE INDEX idx_perf_worker ON worker_performances(worker_id);

CREATE TABLE worker_commissions (
    id                BIGSERIAL PRIMARY KEY,
    worker_id         BIGINT NOT NULL REFERENCES workers(id),
    group_id          BIGINT NOT NULL REFERENCES worker_groups(id),
    group_name        VARCHAR(64) NOT NULL,
    legal_entity_id   BIGINT NOT NULL,
    legal_entity_name VARCHAR(128) NOT NULL,
    region_id         INTEGER NOT NULL,
    region_name       VARCHAR(64) NOT NULL,
    period            VARCHAR(16) NOT NULL,
    formula           VARCHAR(64) NOT NULL,   -- 计提公式
    amount            NUMERIC(12,2) NOT NULL,
    CONSTRAINT uq_worker_comm UNIQUE (worker_id, period, group_id, region_id)
);
CREATE INDEX idx_comm_worker ON worker_commissions(worker_id);

CREATE TABLE worker_schedules (
    id                BIGSERIAL PRIMARY KEY,
    worker_id         BIGINT NOT NULL REFERENCES workers(id),
    group_id          BIGINT NOT NULL REFERENCES worker_groups(id),
    group_name        VARCHAR(64) NOT NULL,
    legal_entity_id   BIGINT NOT NULL,
    legal_entity_name VARCHAR(128) NOT NULL,
    region_id         INTEGER NOT NULL,
    region_name       VARCHAR(64) NOT NULL,
    month             VARCHAR(16) NOT NULL,   -- 2026-08
    busy_days         SMALLINT NOT NULL,
    CONSTRAINT uq_worker_sched UNIQUE (worker_id, month, group_id, region_id)
);
CREATE INDEX idx_sched_worker ON worker_schedules(worker_id);

COMMIT;
