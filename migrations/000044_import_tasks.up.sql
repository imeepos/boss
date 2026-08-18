-- 数据导入任务记录(admin-positioning P2):地址/地理(后续客户/端口/资产)每次导入落一条,
-- 供"数据导入中心"页查询"谁在什么时候导了多少"。
BEGIN;

CREATE TABLE import_tasks (
    id          BIGSERIAL PRIMARY KEY,
    kind        VARCHAR(32) NOT NULL,             -- addresses / geo / ...
    operator_id BIGINT NOT NULL REFERENCES accounts(id),
    imported    INTEGER NOT NULL DEFAULT 0,       -- 成功行数
    failed      INTEGER NOT NULL DEFAULT 0,       -- 失败行数(预留)
    detail      JSONB,                            -- 明细(如 geo 各子计数)
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_import_tasks_time ON import_tasks (created_at DESC);

COMMIT;
