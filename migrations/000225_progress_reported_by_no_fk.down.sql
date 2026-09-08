-- 000225 down: 回到 accounts 单代次语义,WORKER 代次行随语义作废先清。
BEGIN;
DELETE FROM construction_progress WHERE reporter_type = 'WORKER';
ALTER TABLE construction_progress ADD CONSTRAINT construction_progress_reported_by_fkey
    FOREIGN KEY (reported_by) REFERENCES accounts(id);
COMMIT;