-- 数据备份迁移(SYS 域运维工具,迁移 000095):备份/恢复任务记录表 + menu:backup 权限。
BEGIN;
CREATE TABLE IF NOT EXISTS backup_jobs (
    id          BIGSERIAL PRIMARY KEY,
    kind        TEXT NOT NULL,                  -- backup(导出归档) / restore(导入恢复)
    scope       TEXT NOT NULL DEFAULT 'tables', -- tables(选表) / all(public 全表)
    tables      TEXT[] NOT NULL DEFAULT '{}',
    status      TEXT NOT NULL,                  -- running / succeeded / failed
    file_name   TEXT NOT NULL DEFAULT '',
    size_bytes  BIGINT NOT NULL DEFAULT 0,
    table_count INT NOT NULL DEFAULT 0,
    row_count   BIGINT NOT NULL DEFAULT 0,
    error       TEXT NOT NULL DEFAULT '',
    operator_id BIGINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_backup_jobs_created_at ON backup_jobs (created_at DESC);
COMMIT;

-- 菜单权限(menu:backup,与 web/admin menu.def key 一一对应),授予 sysadmin。
BEGIN;
INSERT INTO permissions (code, name) VALUES
    ('menu:backup', '基础配置·数据备份迁移')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:backup'
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;
COMMIT;
