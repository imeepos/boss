-- 招商引资/合作入驻:企业自助申请 → 后台审核 → 开通企业管理账号(对标 000051 客户 onboarding)。
-- 设计:申请与正式组织解耦——待审核企业不入 legal_entities/accounts;
-- 审核通过时建 legal_entities 子公司 + partner_admin 管理账号(绑定 legal_entity_id 做数据隔离)。
-- 字段权威:docs/contract/fields.md(本迁移同步登记)。
BEGIN;

CREATE TABLE partner_applications (
    id                  BIGSERIAL PRIMARY KEY,
    company_name        VARCHAR(128) NOT NULL,                       -- 企业名称
    credit_code         VARCHAR(64)  NOT NULL,                       -- 统一社会信用码
    contact_name        VARCHAR(64)  NOT NULL,                       -- 联系人
    contact_phone       VARCHAR(32)  NOT NULL,                       -- 联系电话
    email               VARCHAR(128),                                -- 邮箱(可空)
    business_desc       TEXT         NOT NULL,                       -- 合作意向说明
    status              VARCHAR(16)  NOT NULL DEFAULT 'PENDING',     -- PENDING 待审核 / APPROVED 已通过 / REJECTED 已驳回
    review_note         TEXT,                                        -- 审核意见(驳回必填)
    reviewer_account_id BIGINT,                                       -- 审核人账号 id(accounts)
    legal_entity_id     BIGINT,                                       -- 审核通过后创建的 legal_entities.id(空=未开通)
    admin_account_id    BIGINT,                                       -- 审核通过后创建的企业管理员账号 id(空=未开通)
    submitted_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    reviewed_at         TIMESTAMPTZ                                   -- null=未审核
);
CREATE INDEX idx_partner_applications_status ON partner_applications(status);
CREATE INDEX idx_partner_applications_credit_code ON partner_applications(credit_code);

-- 入驻企业角色:partner_admin 企业管理员(企业工作台全量);partner_staff 企业员工(只读)。
INSERT INTO roles (code, name) VALUES
    ('partner_admin', '入驻企业管理员'),
    ('partner_staff', '入驻企业员工')
ON CONFLICT (code) DO NOTHING;

-- 菜单权限:menu:partner 后台审核页(sysadmin);企业工作台三页(partner_admin/partner_staff)。
-- 前端 menu.def key 与权限码一一对应(见 web/admin/src/router/menu.def.ts)。
INSERT INTO permissions (code, name) VALUES
    ('menu:partner',        '组织与权限·入驻申请审核'),
    ('menu:partner-home',   '企业工作台·我的企业'),
    ('menu:partner-staff',  '企业工作台·员工管理'),
    ('menu:partner-orders', '企业工作台·企业订单')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('menu:partner')
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('menu:partner-home', 'menu:partner-staff', 'menu:partner-orders')
WHERE r.code = 'partner_admin'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('menu:partner-home', 'menu:partner-orders')
WHERE r.code = 'partner_staff'
ON CONFLICT DO NOTHING;

COMMIT;
