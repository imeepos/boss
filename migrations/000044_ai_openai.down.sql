-- 回滚 000044:摘除 menu:ai 权限及角色绑定;移除 ai.openai.* 配置键(不含业务数据)。
BEGIN;
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:ai');
DELETE FROM permissions WHERE code = 'menu:ai';
DELETE FROM biz_params WHERE key IN ('ai.openai.apiUrl', 'ai.openai.apiKey', 'ai.openai.model');
COMMIT;
