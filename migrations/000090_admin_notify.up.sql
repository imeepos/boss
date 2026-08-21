-- 后台提醒中心(notify 域,docs/plan/admin-notify-center.md):admin 通知+待办。
-- 广播+读回执模型:admin_notifications 面向角色广播,admin_notification_reads 每账号已读。
BEGIN;
CREATE TABLE admin_notifications (
    id           BIGSERIAL PRIMARY KEY,
    category     TEXT NOT NULL,                       -- todo | task
    level        TEXT NOT NULL,                       -- INFO | WARN | URGENT(复用 terms.md 消息 level)
    title        TEXT NOT NULL,
    content      TEXT NOT NULL DEFAULT '',
    link         TEXT NOT NULL DEFAULT '',            -- 前端路由,如 /boss/worker-reg
    ref_type     TEXT NOT NULL DEFAULT '',            -- 来源域标识: importer | worker_reg | provision | report | billing | complaint
    ref_id       TEXT NOT NULL DEFAULT '',            -- 来源域主键(字符串,跨域兼容)
    target_role  TEXT NOT NULL DEFAULT '',            -- 空=全部后台角色;否则 RoleCode
    resolved     BOOLEAN NOT NULL DEFAULT FALSE,      -- todo 专用:来源域完成后置 true
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_admin_notif_feed ON admin_notifications (target_role, created_at DESC);
CREATE INDEX idx_admin_notif_ref ON admin_notifications (ref_type, ref_id);

CREATE TABLE admin_notification_reads (
    notification_id BIGINT NOT NULL REFERENCES admin_notifications(id) ON DELETE CASCADE,
    account_id      BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    read_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (notification_id, account_id)
);
COMMIT;
