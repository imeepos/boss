-- 阶段7 债务偿还修复:provision_logs.result 需承载 "FAILED: <原因>" 留痕,
-- 原 VARCHAR(16) 过短(真实 PG 集成测试触发 SQLSTATE 22001),加宽为 VARCHAR(255)。
BEGIN;

ALTER TABLE provision_logs ALTER COLUMN result TYPE VARCHAR(255);

COMMIT;
