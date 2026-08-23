DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE code = 'menu:openplat');
DELETE FROM permissions WHERE code = 'menu:openplat';
DROP TABLE IF EXISTS open_usage_day;
DROP TABLE IF EXISTS open_webhook_subscriptions;
DROP TABLE IF EXISTS open_apps;
