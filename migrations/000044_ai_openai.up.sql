-- AI 能力网关:OpenAI(openai-go SDK)统一接口,apiKey/apiUrl 由 admin 集中配置。
-- 配置三键落在 biz_params(热更);另加 menu:ai 权限保护配置与调用接口,授予 sysadmin。
BEGIN;

INSERT INTO biz_params(key, value, description) VALUES
    ('ai.openai.apiUrl', '""', 'OpenAI 兼容服务地址,需含 /v1,如 https://api.openai.com/v1'),
    ('ai.openai.apiKey', '""', 'OpenAI 平台级密钥(仅 admin 可见明文,接口返回脱敏)'),
    ('ai.openai.model',  '"gpt-4o-mini"', 'OpenAI 默认模型,请求未指定 model 时兜底')
ON CONFLICT (key) DO NOTHING;

INSERT INTO permissions (code, name) VALUES
    ('menu:ai', 'AI 能力·OpenAI 配置与调用')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:ai'
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;
COMMIT;
