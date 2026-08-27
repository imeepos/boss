package cms

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

var ts = time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)

func postCols() []string {
	return []string{"id", "slug", "lang", "title", "category", "summary", "cover_attachment_id",
		"content", "status", "published_at", "version", "author_name", "updated_at"}
}

func newMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { mock.Close() })
	return mock
}

func row(id int64, slug, status string) *pgxmock.Rows {
	return pgxmock.NewRows(postCols()).AddRow(id, slug, "zh-CN", "标题", "NEWS", "摘要", int64(0),
		"正文", status, nil, 1, nil, "2026-08-28 10:00")
}

// TestValidate 契约:非法 slug/超长 title/非法枚举/空正文一律 ErrInvalidPost。
func TestValidate(t *testing.T) {
	base := func() Post {
		return Post{Title: "t", Slug: "hello-world", Category: "NEWS", Lang: LangDefault,
			Summary: "s", Content: "c", Status: StatusDraft}
	}
	cases := []func(*Post){
		func(p *Post) { p.Slug = "Bad_Slug" },
		func(p *Post) { p.Slug = "-leading" },
		func(p *Post) { p.Title = "" },
		func(p *Post) { p.Category = "blog" }, // 小写非法 code;存在性/启用另由 categoryUsable 拦
		func(p *Post) { p.Status = "SCHEDULED" },
		func(p *Post) { p.Content = "" },
		func(p *Post) { p.Lang = "fr-FR" }, // 语言集外非法(000155)
	}
	for i, mutate := range cases {
		p := base()
		mutate(&p)
		if !errors.Is(p.validate(), ErrInvalidPost) {
			t.Fatalf("case %d: want ErrInvalidPost", i)
		}
	}
	p := base()
	if err := p.validate(); err != nil {
		t.Fatalf("valid post rejected: %v", err)
	}
}

// TestPGStore_ListPublished 契约:仅 PUBLISHED、按语言过滤、发布时间倒序,lang/category/limit 依序注入。
func TestPGStore_ListPublished(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(`WHERE status='PUBLISHED' AND lang=\$1`).
		WithArgs("zh-CN", "NEWS", 10).
		WillReturnRows(row(1, "a", StatusPublished))

	s := NewPGStore(mock)
	got, err := s.ListPublished(context.Background(), "NEWS", "zh-CN", 10)
	if err != nil || len(got) != 1 || got[0].Slug != "a" || got[0].Lang != "zh-CN" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestPGStore_GetPublishedBySlug_HidesDraft 契约:草稿/下线一律 ErrPostNotFound。
func TestPGStore_GetPublishedBySlug_HidesDraft(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(`WHERE slug=\$1 AND lang=\$2 AND status='PUBLISHED'`).WithArgs("draft-slug", "zh-CN").
		WillReturnError(pgx.ErrNoRows)

	if _, err := NewPGStore(mock).GetPublishedBySlug(context.Background(), "draft-slug", "zh-CN"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("want ErrPostNotFound, got %v", err)
	}
}

// TestPGStore_GetPublishedBySlug_LangFallback 契约:请求语言缺变体回退默认语言(000155)。
func TestPGStore_GetPublishedBySlug_LangFallback(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(`WHERE slug=\$1 AND lang=\$2 AND status='PUBLISHED'`).WithArgs("a", "en-US").
		WillReturnError(pgx.ErrNoRows)
	mock.ExpectQuery(`WHERE slug=\$1 AND lang=\$2 AND status='PUBLISHED'`).WithArgs("a", "zh-CN").
		WillReturnRows(row(1, "a", StatusPublished))

	p, err := NewPGStore(mock).GetPublishedBySlug(context.Background(), "a", "en-US")
	if err != nil || p.Lang != "zh-CN" {
		t.Fatalf("want zh-CN fallback, got %+v err=%v", p, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// expectCatUsable 桩掉 Create/Update 前的分类存在+启用预检。
func expectCatUsable(mock pgxmock.PgxPoolIface, code string) {
	mock.ExpectQuery(`SELECT enabled FROM cms_categories`).
		WithArgs(code).WillReturnRows(pgxmock.NewRows([]string{"enabled"}).AddRow(true))
}

// TestPGStore_CreatePost_SlugTaken 契约:slug+lang 唯一冲突映射 ErrSlugTaken。
func TestPGStore_CreatePost_SlugTaken(t *testing.T) {
	mock := newMock(t)
	expectCatUsable(mock, "NEWS")
	mock.ExpectQuery(`INSERT INTO cms_posts`).WithArgs(
		"dup", "zh-CN", "t", "NEWS", "s", nil, "c", StatusDraft, "").
		WillReturnError(&pgconn.PgError{Code: "23505"})

	if _, err := NewPGStore(mock).CreatePost(context.Background(),
		Post{Slug: "dup", Title: "t", Summary: "s", Content: "c"}); !errors.Is(err, ErrSlugTaken) {
		t.Fatalf("want ErrSlugTaken, got %v", err)
	}
}

// TestPGStore_CreatePost_PublishedAtOnce 契约:创建即 PUBLISHED 也落发布时间(102 回放发现的缺陷)。
func TestPGStore_CreatePost_PublishedAtOnce(t *testing.T) {
	mock := newMock(t)
	expectCatUsable(mock, "NEWS")
	mock.ExpectQuery(`INSERT INTO cms_posts`).WithArgs(
		"go-live", "en-US", "t", "NEWS", "s", nil, "c", StatusPublished, "a").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(2)))

	if _, err := NewPGStore(mock).CreatePost(context.Background(),
		Post{Slug: "go-live", Lang: "en-US", Title: "t", Summary: "s", Content: "c",
			Status: StatusPublished, AuthorName: "a"}); err != nil {
		t.Fatalf("create published: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestPGStore_UpdatePost_SetsPublishedAt 契约:置 PUBLISHED 且从未发布时落 now()。
func TestPGStore_UpdatePost_SetsPublishedAt(t *testing.T) {
	mock := newMock(t)
	expectCatUsable(mock, "NEWS")
	mock.ExpectExec(`UPDATE cms_posts`).WithArgs(
		int64(1), "a", "zh-CN", "t", "NEWS", "s", nil, "c", StatusPublished, "").
		WillReturnResult(pgconn.NewCommandTag("UPDATE 1"))

	if err := NewPGStore(mock).UpdatePost(context.Background(),
		Post{ID: 1, Slug: "a", Title: "t", Summary: "s", Content: "c", Status: StatusPublished}); err != nil {
		t.Fatalf("update: %v", err)
	}
}
