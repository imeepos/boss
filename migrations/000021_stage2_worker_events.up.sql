-- 阶段2:师傅事件事实表(物料领用/工具借用/服务评价/资产归还)。
-- 字段权威:server-ts/src/entities/worker.ts。
BEGIN;

CREATE TABLE worker_materials (
    id                BIGSERIAL PRIMARY KEY,
    worker_id         BIGINT NOT NULL REFERENCES workers(id),
    group_id          BIGINT NOT NULL REFERENCES worker_groups(id),
    group_name        VARCHAR(64) NOT NULL,
    legal_entity_id   BIGINT NOT NULL,
    legal_entity_name VARCHAR(128) NOT NULL,
    region_id         INTEGER NOT NULL,
    region_name       VARCHAR(64) NOT NULL,
    name              VARCHAR(64) NOT NULL,
    qty               INTEGER NOT NULL
);
CREATE INDEX idx_materials_worker ON worker_materials(worker_id);

CREATE TABLE worker_tools (
    id                BIGSERIAL PRIMARY KEY,
    worker_id         BIGINT NOT NULL REFERENCES workers(id),
    group_id          BIGINT NOT NULL REFERENCES worker_groups(id),
    group_name        VARCHAR(64) NOT NULL,
    legal_entity_id   BIGINT NOT NULL,
    legal_entity_name VARCHAR(128) NOT NULL,
    region_id         INTEGER NOT NULL,
    region_name       VARCHAR(64) NOT NULL,
    name              VARCHAR(64) NOT NULL,
    borrowed          BOOLEAN NOT NULL
);
CREATE INDEX idx_tools_worker ON worker_tools(worker_id);

CREATE TABLE worker_feedbacks (
    id                BIGSERIAL PRIMARY KEY,
    worker_id         BIGINT NOT NULL REFERENCES workers(id),
    worker_name       VARCHAR(64) NOT NULL,
    group_id          BIGINT NOT NULL REFERENCES worker_groups(id),
    group_name        VARCHAR(64) NOT NULL,
    legal_entity_id   BIGINT NOT NULL,
    legal_entity_name VARCHAR(128) NOT NULL,
    region_id         INTEGER NOT NULL,
    region_name       VARCHAR(64) NOT NULL,
    ticket_id         BIGINT NOT NULL,
    customer_id       BIGINT NOT NULL,
    customer_name     VARCHAR(64) NOT NULL,
    score             SMALLINT NOT NULL,
    need_review       BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX idx_feedbacks_worker ON worker_feedbacks(worker_id);

CREATE TABLE asset_returns (
    id                BIGSERIAL PRIMARY KEY,
    worker_id         BIGINT NOT NULL REFERENCES workers(id),
    group_id          BIGINT NOT NULL REFERENCES worker_groups(id),
    group_name        VARCHAR(64) NOT NULL,
    legal_entity_id   BIGINT NOT NULL,
    legal_entity_name VARCHAR(128) NOT NULL,
    region_id         INTEGER NOT NULL,
    region_name       VARCHAR(64) NOT NULL,
    asset_id          BIGINT NOT NULL,
    reason            VARCHAR(128) NOT NULL,
    status            VARCHAR(16) NOT NULL DEFAULT 'PENDING'
);
CREATE INDEX idx_returns_worker ON asset_returns(worker_id);

COMMIT;
