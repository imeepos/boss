-- Q2 P1 待办处理时限:admin_notifications 补 due_at(时限)/resolved_at(办结时间)。
-- Emit 对 todo 可带 DueHours(如四码冲突 4 小时),Resolve 记 resolved_at,
-- 巡检循环对超时未办就地升级 URGENT,SLAStats 出按时办结率。
BEGIN;

ALTER TABLE admin_notifications
    ADD COLUMN due_at     TIMESTAMPTZ,
    ADD COLUMN resolved_at TIMESTAMPTZ;

CREATE INDEX idx_admin_notif_sla ON admin_notifications(category, resolved, due_at);

COMMIT;
