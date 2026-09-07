-- 000224: 施工进度上报上报人代次(P-INFRA-1 W7,F5a):
-- 进度记录只增不改;上报人限管理账号或师傅,reporter_type 区分代次,
-- reported_by 语义=reporter_type 域内 id(ACCOUNT->accounts.id / WORKER->workers.id),
-- 存量行(000213 起)全部为 ACCOUNT,零迁移负担。
BEGIN;
ALTER TABLE construction_progress ADD COLUMN reporter_type VARCHAR(8) NOT NULL DEFAULT 'ACCOUNT';
CREATE INDEX idx_construction_progress_points ON construction_progress (project_id) WHERE lat IS NOT NULL;
COMMIT;