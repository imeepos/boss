-- 官网内容多语言(cms):cms_posts 加 lang 语言维度,同 slug 多语言变体共用 URL,
-- 唯一键从 slug 全表唯一改为 (slug, lang) 组合唯一;cms_categories 加 name_i18n
-- JSONB 多语言名(name 保留为默认/回退)。存量数据统一归 zh-CN;
-- 公开读按 lang 精确匹配,缺变体回退 zh-CN(域层 GetPublishedBySlug 兜底)。
BEGIN;

ALTER TABLE cms_posts ADD COLUMN lang VARCHAR(8) NOT NULL DEFAULT 'zh-CN';

ALTER TABLE cms_posts DROP CONSTRAINT cms_posts_slug_key;
ALTER TABLE cms_posts ADD CONSTRAINT cms_posts_slug_lang_key UNIQUE (slug, lang);

ALTER TABLE cms_categories ADD COLUMN name_i18n JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMIT;
