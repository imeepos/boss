-- API key 权限模板：受限密钥仅能使用账号角色已有权限的子集。
BEGIN;
ALTER TABLE api_keys ADD COLUMN template_code VARCHAR(64);
CREATE TABLE api_key_permission_templates (
    code VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE api_key_template_permissions (
    template_code VARCHAR(64) NOT NULL REFERENCES api_key_permission_templates(code) ON DELETE CASCADE,
    permission_code VARCHAR(128) NOT NULL,
    PRIMARY KEY (template_code, permission_code)
);
INSERT INTO api_key_permission_templates(code, name, description) VALUES
 ('partner-orders-read', '伙伴订单只读', '仅访问企业订单查询接口')
ON CONFLICT (code) DO NOTHING;
INSERT INTO api_key_template_permissions(template_code, permission_code)
VALUES ('partner-orders-read', 'menu:partner-orders') ON CONFLICT DO NOTHING;
COMMIT;
