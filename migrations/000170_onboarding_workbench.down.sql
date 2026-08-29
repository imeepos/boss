-- 回滚 000170:撤销 menu:onboarding 权限码及授权。
BEGIN;
DELETE FROM role_permissions WHERE permission_id IN
  (SELECT id FROM permissions WHERE code = 'menu:onboarding');
DELETE FROM permissions WHERE code = 'menu:onboarding';
COMMIT;
