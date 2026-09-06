-- ODN P6 施工项目与竣工回填(路线图 T9):施工单 + 单-设施明细 + as-built 竣工信息。
-- 状态机 PENDING→BUILDING→ACCEPTED(线性);ACCEPTED 时批量把单内设施 lifecycle 推到 IN_SERVICE。
BEGIN;

CREATE TABLE construction_projects (
    id           BIGSERIAL PRIMARY KEY,
    proj_no      VARCHAR(32)  NOT NULL UNIQUE,
    name         VARCHAR(128) NOT NULL DEFAULT '',
    prv_code     CHAR(6),
    city_prefix  VARCHAR(5),
    status       VARCHAR(16)  NOT NULL DEFAULT 'PENDING'
                 CHECK (status IN ('PENDING','BUILDING','ACCEPTED')),
    asbuilt_note VARCHAR(255) NOT NULL DEFAULT '',
    accepted_by  BIGINT REFERENCES accounts(id),
    accepted_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (prv_code, city_prefix) REFERENCES odn_city_code (prv_code, city_prefix)
);

CREATE TABLE construction_items (
    id            BIGSERIAL PRIMARY KEY,
    project_id    BIGINT      NOT NULL REFERENCES construction_projects(id) ON DELETE CASCADE,
    facility_code VARCHAR(8)  NOT NULL REFERENCES odn_facility(code),
    CONSTRAINT construction_items_uq UNIQUE (project_id, facility_code)
);
CREATE INDEX idx_construction_items_proj ON construction_items (project_id);

COMMIT;
