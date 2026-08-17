-- 阶段2:师傅台账(班组归属台账 / 接单设置 / 站内消息)。
-- 字段权威:docs/contract/fields.md §7 + server-ts/src/entities/worker.ts。
BEGIN;

CREATE TABLE worker_group_memberships (
    id                  BIGSERIAL PRIMARY KEY,
    worker_id           BIGINT NOT NULL REFERENCES workers(id),
    group_id            BIGINT NOT NULL REFERENCES worker_groups(id),
    group_name          VARCHAR(64) NOT NULL,   -- 班组名快照
    legal_entity_id     BIGINT NOT NULL,        -- 公司级历史归属快照
    legal_entity_name   VARCHAR(128) NOT NULL,
    region_id           INTEGER NOT NULL,       -- 当时服务区域快照
    region_name         VARCHAR(64),
    reason              VARCHAR(128),
    operator_account_id BIGINT,
    effective_from      TIMESTAMPTZ NOT NULL,
    effective_to        TIMESTAMPTZ             -- null=至今
);
CREATE INDEX idx_memberships_worker ON worker_group_memberships(worker_id, effective_from);

CREATE TABLE worker_settings (
    id           BIGSERIAL PRIMARY KEY,
    worker_id    BIGINT NOT NULL UNIQUE REFERENCES workers(id), -- 师傅1:1
    accepting    BOOLEAN NOT NULL DEFAULT true,
    radius_km    SMALLINT NOT NULL,
    accept_types VARCHAR(255) NOT NULL,
    updated_at   TIMESTAMPTZ
);

CREATE TABLE worker_messages (
    id        BIGSERIAL PRIMARY KEY,
    worker_id BIGINT NOT NULL REFERENCES workers(id),
    level     VARCHAR(8) NOT NULL,              -- INFO/WARN/URGENT
    title     VARCHAR(128) NOT NULL,
    content   TEXT NOT NULL,
    sent_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    read      BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX idx_messages_worker ON worker_messages(worker_id);

COMMIT;
