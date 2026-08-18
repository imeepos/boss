-- 用户端数据域菜单权限码种子(userdata.yaml 路由门禁)。
BEGIN;
INSERT INTO permissions (code, name) VALUES
    ('menu:user',     '用户端·用户列表与详情'),
    ('menu:userdata', '用户端·用户数据管理')
ON CONFLICT (code) DO NOTHING;
COMMIT;
