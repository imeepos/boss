DELETE FROM role_permissions
 WHERE permission_id = (SELECT id FROM permissions WHERE code = 'menu:realname-review');
DELETE FROM permissions WHERE code = 'menu:realname-review';
