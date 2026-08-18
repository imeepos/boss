BEGIN;
ALTER TABLE provision_tasks DROP COLUMN task_no;
ALTER TABLE provision_tasks DROP COLUMN order_id;
ALTER TABLE provision_tasks DROP COLUMN stage_event;
COMMIT;
