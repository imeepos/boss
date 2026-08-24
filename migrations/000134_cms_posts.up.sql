-- 官网内容发布域(cms):cms_posts 单表承载动态/新闻/文章(category 区分)。
-- 范式沿用 cs_knowledge_articles(000118):version 自增防并发覆盖,软状态机。
-- 公开读只吐 status=PUBLISHED(见 adopted note 2026-08-28-cms-site-posts)。
BEGIN;

CREATE TABLE cms_posts (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(120) NOT NULL UNIQUE,
    title VARCHAR(160) NOT NULL,
    category VARCHAR(16) NOT NULL DEFAULT 'NEWS' CHECK (category IN ('NEWS', 'ARTICLE')),
    summary VARCHAR(500) NOT NULL DEFAULT '',
    cover_attachment_id BIGINT REFERENCES attachments(id),
    content TEXT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'PUBLISHED', 'OFFLINE')),
    published_at TIMESTAMPTZ,
    version INTEGER NOT NULL DEFAULT 1,
    author_name VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 官网首页列表:已发布按发布时间倒序取最新 N 条。
CREATE INDEX idx_cms_posts_public ON cms_posts(status, published_at DESC) WHERE status = 'PUBLISHED';
-- admin 列表默认按 id 倒序。
CREATE INDEX idx_cms_posts_id_desc ON cms_posts(id DESC);

COMMIT;
