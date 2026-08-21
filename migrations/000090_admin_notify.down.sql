-- 回滚后台提醒中心两表。
BEGIN;
DROP TABLE IF EXISTS admin_notification_reads;
DROP TABLE IF EXISTS admin_notifications;
COMMIT;
