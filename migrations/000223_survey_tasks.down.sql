-- 000223 down: 勘测任务域回滚。
BEGIN;
DROP TABLE survey_task_reports;
DROP TABLE survey_tasks;
COMMIT;