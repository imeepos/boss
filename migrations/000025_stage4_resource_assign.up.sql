-- 阶段4:网络资源(OSS)。设备归属台账 + QoS 模板。
-- 字段权威:server-ts/src/entities/oss.ts(ResourceAssignment/QosTemplate)。
BEGIN;

CREATE TABLE resource_assignments (
    id                 BIGSERIAL PRIMARY KEY,
    resource_id        BIGINT NOT NULL REFERENCES resources(id),
    legal_entity_id    BIGINT NOT NULL REFERENCES legal_entities(id),
    legal_entity_name  VARCHAR(128) NOT NULL,
    address_id         BIGINT REFERENCES addresses(id),
    address_name       VARCHAR(64),
    region_id          INTEGER,
    region_name        VARCHAR(64),
    reason             VARCHAR(128),
    operator_account_id BIGINT,
    effective_from     TIMESTAMPTZ NOT NULL,
    effective_to       TIMESTAMPTZ               -- null=至今
);
CREATE INDEX idx_resource_assignments ON resource_assignments(resource_id, effective_from);

CREATE TABLE qos_templates (
    id             BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities(id),
    code           VARCHAR(32) NOT NULL UNIQUE,  -- 如 QoS-VIP
    name           VARCHAR(64) NOT NULL          -- 如 VIP
);

COMMIT;
