-- 阶段7 债务偿还:provision_tasks 对齐 gRPC 契约(api/proto/boss/provision/v1)。
-- task_no 外部稳定标识(EnqueueTask/GetTask/RetryTask 寻址);order_id/stage_event 承载来源订单与环节事件。
BEGIN;

ALTER TABLE provision_tasks ADD COLUMN task_no VARCHAR(32);
ALTER TABLE provision_tasks ADD COLUMN order_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE provision_tasks ADD COLUMN stage_event VARCHAR(32) NOT NULL DEFAULT '';

-- 存量行回填(task_no = 自增 id 派生),保证唯一索引可建。
UPDATE provision_tasks SET task_no = 'TASK-' || id WHERE task_no IS NULL;
ALTER TABLE provision_tasks ALTER COLUMN task_no SET NOT NULL;
CREATE UNIQUE INDEX idx_provision_tasks_task_no ON provision_tasks(task_no);
CREATE INDEX idx_provision_tasks_order ON provision_tasks(order_id);

COMMIT;
