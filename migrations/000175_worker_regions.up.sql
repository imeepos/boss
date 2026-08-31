-- 师傅多负责区域(2026-09-01 后台需求:一个师傅可配置多个负责区域)。
-- workers.region_id 保留为主区域(第一负责区域,兼容派单快照/月度统计链路),
-- 扩展区域落 worker_regions;权威字段:docs/contract/fields.md §7.1(本迁移同步登记)。
BEGIN;

CREATE TABLE worker_regions (
    worker_id BIGINT  NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    region_id INTEGER NOT NULL REFERENCES regions(id),
    PRIMARY KEY (worker_id, region_id)
);
CREATE INDEX idx_worker_regions_region ON worker_regions(region_id);

-- 存量种子:主区域视为首个负责区域,历史师傅多区域语义无缝衔接。
INSERT INTO worker_regions(worker_id, region_id)
SELECT id, region_id FROM workers WHERE region_id > 0
ON CONFLICT DO NOTHING;

COMMIT;
