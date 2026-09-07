-- 000214: 施工质量记录(P0-C):资源级测试证据与整改闭环。
-- 测试记录 append-only(每次读数即事实);整改项 OPEN→RECTIFYING→VERIFIED,
-- ACCEPTED 前置:同项目不存在未 VERIFIED 整改项(服务端事务内校验)。
CREATE TABLE construction_tests (
    id            BIGSERIAL PRIMARY KEY,
    project_id    BIGINT NOT NULL REFERENCES construction_projects(id),
    resource_type VARCHAR(16) NOT NULL CHECK (resource_type IN ('FACILITY','SEGMENT','FIBER','PORT')),
    resource_ref  VARCHAR(64) NOT NULL,
    test_kind     VARCHAR(16) NOT NULL CHECK (test_kind IN ('OTDR','OPTICAL_POWER','CONNECTIVITY')),
    result        VARCHAR(8) NOT NULL CHECK (result IN ('PASS','FAIL')),
    attenuation_db NUMERIC(6,2),
    power_dbm      NUMERIC(6,2),
    note           TEXT,
    reported_by    BIGINT REFERENCES accounts,
    reported_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_construction_tests_project ON construction_tests (project_id, resource_type, resource_ref);

CREATE TABLE construction_defects (
    id            BIGSERIAL PRIMARY KEY,
    project_id    BIGINT NOT NULL REFERENCES construction_projects(id),
    facility_code VARCHAR(8) NOT NULL REFERENCES odn_facility(code),
    severity      VARCHAR(8) NOT NULL CHECK (severity IN ('MINOR','MAJOR','CRITICAL')),
    description   VARCHAR(255) NOT NULL,
    status        VARCHAR(10) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN','RECTIFYING','VERIFIED')),
    note          TEXT,
    opened_by     BIGINT REFERENCES accounts,
    opened_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    rectified_at  TIMESTAMPTZ,
    verified_by   BIGINT REFERENCES accounts,
    verified_at   TIMESTAMPTZ
);
CREATE INDEX idx_construction_defects_project ON construction_defects (project_id, status);
