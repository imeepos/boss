-- 回滚前提:库内无 NEWS/ARTICLE 之外的自定义分类被 posts 引用,否则先人工清理。
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'menu:site-cats');
DELETE FROM permissions WHERE code = 'menu:site-cats';
ALTER TABLE cms_posts DROP CONSTRAINT cms_posts_category_fk;
ALTER TABLE cms_posts ALTER COLUMN category TYPE VARCHAR(16);
ALTER TABLE cms_posts ADD CONSTRAINT cms_posts_category_check CHECK (category IN ('NEWS', 'ARTICLE'));
DROP TABLE cms_categories;
