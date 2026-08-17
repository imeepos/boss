-- 阶段2:师傅域(班组 + 师傅)。
-- 字段权威:docs/contract/fields.md §7 + server-ts/src/entities/org.ts(WorkerGroup)/worker.ts(Worker)。
BEGIN;

CREATE TABLE worker_groups (
    id              BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities(id),
    code            VARCHAR(32) NOT NULL,    -- 班组编码,公司内唯一(稳定标识)
    name            VARCHAR(64) NOT NULL,
    leader_id       BIGINT,                  -- 组长,软引用 workers(可空)
    leader_name     VARCHAR(64),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_worker_groups_entity_code UNIQUE (legal_entity_id, code)
);

CREATE TABLE workers (
    id        BIGSERIAL PRIMARY KEY,
    staff_no  VARCHAR(32) NOT NULL UNIQUE,   -- 工号 WK-1024
    name      VARCHAR(64) NOT NULL,
    group_id  BIGINT NOT NULL REFERENCES worker_groups(id),
    region_id INTEGER NOT NULL,              -- 服务区域快照
    phone     VARCHAR(32) NOT NULL,
    status    SMALLINT NOT NULL DEFAULT 1,   -- 1在职 0离职
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at   TIMESTAMPTZ                    -- null=在职
);
CREATE INDEX idx_workers_group ON workers(group_id);

COMMIT;
