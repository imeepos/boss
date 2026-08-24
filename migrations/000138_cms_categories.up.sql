-- 官网内容分类自定义域(cms_categories):分类从 NEWS/ARTICLE 硬编码改为可管理字典表。
-- cms_posts.category 去枚举 CHECK,加宽到 32 并软引用 cms_categories.code(域层校验存在+启用)。
-- 同迁移登记菜单权限 menu:site-cats(分类管理页,菜单 key=site-cats,遵循"权限码迁移随菜单走")。
BEGIN;

CREATE TABLE cms_categories (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(32) NOT NULL UNIQUE,
    name VARCHAR(64) NOT NULL,
    sort_no INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 存量两分类迁入字典(已在库的 posts.category 值域不破坏)。
INSERT INTO cms_categories (code, name, sort_no) VALUES
    ('NEWS', '动态', 0),
    ('ARTICLE', '文章', 1);

ALTER TABLE cms_posts DROP CONSTRAINT cms_posts_category_check;
ALTER TABLE cms_posts ALTER COLUMN category TYPE VARCHAR(32);
ALTER TABLE cms_posts ADD CONSTRAINT cms_posts_category_fk
    FOREIGN KEY (category) REFERENCES cms_categories(code) ON UPDATE CASCADE;

INSERT INTO permissions (code, name) VALUES
    ('menu:site-cats', '订单与工单·官网分类')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'menu:site-cats'
WHERE r.code = 'sysadmin'
ON CONFLICT DO NOTHING;

COMMIT;
