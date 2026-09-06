-- ODN P6 设施状态机(路线图 T8,docs/plan/odn-lifecycle-roadmap.md):生命周期与在用性(status)分离。
-- 存量默认 IN_SERVICE 兼容;RETIRED 时两列由域层同步;转移规则见 internal/domain/odn/lifecycle.go。
BEGIN;

ALTER TABLE odn_facility ADD COLUMN lifecycle_status VARCHAR(16) NOT NULL DEFAULT 'IN_SERVICE'
    CHECK (lifecycle_status IN ('PLANNED','IN_BUILD','IN_SERVICE','RETIRED'));
ALTER TABLE odn_site ADD COLUMN lifecycle_status VARCHAR(16) NOT NULL DEFAULT 'IN_SERVICE'
    CHECK (lifecycle_status IN ('PLANNED','IN_BUILD','IN_SERVICE','RETIRED'));
ALTER TABLE odn_device ADD COLUMN lifecycle_status VARCHAR(16) NOT NULL DEFAULT 'IN_SERVICE'
    CHECK (lifecycle_status IN ('PLANNED','IN_BUILD','IN_SERVICE','RETIRED'));

COMMIT;
