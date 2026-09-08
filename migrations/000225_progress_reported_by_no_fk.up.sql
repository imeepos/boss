-- 000225: 进度上报人代次落地收尾(W7,000224 漏项)。
-- reported_by 语义随 reporter_type 代次(ACCOUNT=accounts.id / WORKER=workers.id),
原 accounts 外键对 WORKER 代次必然 23503,改为应用层校验。
BEGIN;
ALTER TABLE construction_progress DROP CONSTRAINT construction_progress_reported_by_fkey;
COMMIT;