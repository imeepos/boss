-- 回滚 000155:还原 slug 全表唯一、删 lang 与 name_i18n。
-- 注意:若已存在同 slug 多语言变体,还原唯一键会因重复 slug 失败(需先清理变体数据)。
BEGIN;

ALTER TABLE cms_posts DROP CONSTRAINT IF EXISTS cms_posts_slug_lang_key;
ALTER TABLE cms_posts ADD CONSTRAINT cms_posts_slug_key UNIQUE (slug);
ALTER TABLE cms_posts DROP COLUMN IF EXISTS lang;
ALTER TABLE cms_categories DROP COLUMN IF EXISTS name_i18n;

COMMIT;
