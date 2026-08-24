// Package cms 官网内容发布域:cms_posts 单表承载动态/新闻/文章。
// 契约见 docs/contract/fields.md 8E;裁定见 docs/notes/adopted/2026-08-28-cms-site-posts.md。
package cms

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strconv"
)

// 状态机(terms.md):DRAFT→PUBLISHED 落 published_at;OFFLINE 下线留数据。
const (
	StatusDraft     = "DRAFT"
	StatusPublished = "PUBLISHED"
	StatusOffline   = "OFFLINE"
)

// 分类已改字典表(cms_categories,000137);NEWS/ARTICLE 为迁移种子值,
// 不再作枚举常量校验,存在性+启用由 store 层 categoryUsable 保证。

var (
	// ErrPostNotFound 文章不存在(或公开读时非 PUBLISHED,统一 404 不泄露状态)。
	ErrPostNotFound = errors.New("cms: post not found")
	// ErrSlugTaken slug 已被占用。
	ErrSlugTaken = errors.New("cms: slug already taken")
	// ErrInvalidPost 字段校验失败(title/slug/枚举)。
	ErrInvalidPost = errors.New("cms: invalid post")
)

var slugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Post cms_posts 行投影;时间为展示格式字符串(与 cs 域一致)。
type Post struct {
	ID              int64  `json:"id"`
	Slug            string `json:"slug"`
	Title           string `json:"title"`
	Category        string `json:"category"`
	Summary         string `json:"summary"`
	CoverAttachment int64  `json:"coverAttachmentId"`
	Content         string `json:"content"`
	Status          string `json:"status"`
	PublishedAt     string `json:"publishedAt"`
	Version         int    `json:"version"`
	AuthorName      string `json:"authorName"`
	UpdatedAt       string `json:"updatedAt"`
}

// validate 落库前校验;category/status 空时由调用方给默认值后再校验。
func (p *Post) validate() error {
	if p.Title == "" || len(p.Title) > 160 {
		return ErrInvalidPost
	}
	if !slugRe.MatchString(p.Slug) || len(p.Slug) > 120 {
		return ErrInvalidPost
	}
	if !codeRe.MatchString(p.Category) {
		return ErrInvalidPost
	}
	if p.Status != StatusDraft && p.Status != StatusPublished && p.Status != StatusOffline {
		return ErrInvalidPost
	}
	if len(p.Summary) > 500 || p.Content == "" {
		return ErrInvalidPost
	}
	return nil
}

// Service 域接口:admin 管理面 + 官网匿名只读面 + 分类字典面。
type Service interface {
	ListPosts(ctx context.Context) ([]Post, error)
	ListPublished(ctx context.Context, category string, limit int) ([]Post, error)
	GetPublishedBySlug(ctx context.Context, slug string) (*Post, error)
	CreatePost(ctx context.Context, p Post) (int64, error)
	UpdatePost(ctx context.Context, p Post) error
	DeletePost(ctx context.Context, id int64) error
	CategoryStore
}

// attRefRe 正文内附件引用:Markdown 图片/链接目标 `](att/<id>)`。
// 公开读时重写为 /site/posts/:slug/img/:id,编辑器预览走 admin 附件内容端点。
var attRefRe = regexp.MustCompile(`\]\(att/([0-9]+)\)`)

// AttRefs 列出正文引用的附件 id(去重,升序),公开图片端点据此校验引用关系。
func AttRefs(content string) []int64 {
	seen := map[int64]bool{}
	var out []int64
	for _, m := range attRefRe.FindAllStringSubmatch(content, -1) {
		id, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil || id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// RewriteAttRefs 把 `](att/N)` 目标重写为 urlFor(N);未匹配原样返回。
func RewriteAttRefs(content string, urlFor func(id int64) string) string {
	return attRefRe.ReplaceAllStringFunc(content, func(m string) string {
		sub := attRefRe.FindStringSubmatch(m)
		id, err := strconv.ParseInt(sub[1], 10, 64)
		if err != nil || id <= 0 {
			return m
		}
		return `](` + urlFor(id) + `)`
	})
}
