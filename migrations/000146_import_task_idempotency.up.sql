-- 导入任务客户端幂等键：网络重试或重复登记只保留一条逻辑任务记录。
BEGIN;
ALTER TABLE import_tasks ADD COLUMN client_key VARCHAR(96);
CREATE UNIQUE INDEX uq_import_tasks_client_key
  ON import_tasks (client_key) WHERE client_key IS NOT NULL;
COMMIT;
