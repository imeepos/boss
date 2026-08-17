-- 阶段1:基础平台与基础数据
BEGIN;

CREATE EXTENSION IF NOT EXISTS ltree;

-- 账号 / 角色 / 权限(RBAC,权限变更即时生效=快照走Redis)
CREATE TABLE roles (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(64) NOT NULL UNIQUE,      -- customer/technician/asset_admin/resource_admin/ops/analyst/sysadmin
    name        VARCHAR(128) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE permissions (
    id      BIGSERIAL PRIMARY KEY,
    code    VARCHAR(128) NOT NULL UNIQUE,          -- 如 asset:create
    name    VARCHAR(128) NOT NULL
);

CREATE TABLE role_permissions (
    role_id       BIGINT NOT NULL REFERENCES roles(id),
    permission_id BIGINT NOT NULL REFERENCES permissions(id),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE accounts (
    id            BIGSERIAL PRIMARY KEY,
    username      VARCHAR(64) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    real_name     VARCHAR(64) NOT NULL,
    phone         VARCHAR(32),
    role_id       BIGINT NOT NULL REFERENCES roles(id),
    status        SMALLINT NOT NULL DEFAULT 1,     -- 1启用 0停用
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 审计表:按月分区,可按人/时间/操作类型查询
CREATE TABLE audit_logs (
    id          BIGSERIAL,
    account_id  BIGINT NOT NULL,
    action      VARCHAR(64) NOT NULL,              -- 数据变更/状态变更/权限变更
    target_type VARCHAR(64) NOT NULL,
    target_id   VARCHAR(64),
    detail      JSONB NOT NULL DEFAULT '{}',
    ip          INET,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);
CREATE TABLE audit_logs_2025_08 PARTITION OF audit_logs
    FOR VALUES FROM ('2025-08-01') TO ('2025-09-01');
CREATE INDEX idx_audit_account_time ON audit_logs (account_id, created_at DESC);
CREATE INDEX idx_audit_action ON audit_logs (action, created_at DESC);

-- 业务参数(欠费阈值/预占有效期/核对周期/报告周期),集中配置可调整
CREATE TABLE biz_params (
    key         VARCHAR(128) PRIMARY KEY,
    value       JSONB NOT NULL,
    description VARCHAR(255),
    updated_by  BIGINT REFERENCES accounts(id),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 区域地址层级:市→区→街道→小区→楼栋,ltree 物化路径
CREATE TABLE addresses (
    id          BIGSERIAL PRIMARY KEY,
    path        LTREE NOT NULL,                    -- 如 bj.chaoyang.wangjing.xq1.ld2
    level       SMALLINT NOT NULL,                 -- 1市 2区 3街道 4小区 5楼栋
    name        VARCHAR(128) NOT NULL,
    parent_id   BIGINT REFERENCES addresses(id),
    geom        GEOGRAPHY(POINT),                  -- 阶段8 GIS 预留
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_addresses_path UNIQUE (path)
);
CREATE INDEX idx_addresses_gist ON addresses USING GIST (path);
CREATE INDEX idx_addresses_ggeom ON addresses USING GIST (geom);

COMMIT;
