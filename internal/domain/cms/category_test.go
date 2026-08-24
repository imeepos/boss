package cms

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

func catTestCols() []string {
	return []string{"id", "code", "name", "sort_no", "enabled", "updated_at"}
}

func catRow(id int64, code string, enabled bool) *pgxmock.Rows {
	return pgxmock.NewRows(catTestCols()).AddRow(id, code, "名称", 0, enabled, "2026-08-28 10:00")
}

// TestCategoryValidate 契约:code 大写蛇形 2~32,name 非空 ≤64。
func TestCategoryValidate(t *testing.T) {
	for _, code := range []string{"", "N", "blog", "NEWS_X_超长_________________________", "A-B"} {
		c := &Category{Code: code, Name: "n"}
		if !errors.Is(c.validate(), ErrInvalidCategory) {
			t.Fatalf("code %q should be invalid", code)
		}
	}
	if err := (&Category{Code: "NEWS", Name: "动态"}).validate(); err != nil {
		t.Fatal(err)
	}
}

// TestPGStore_CategoryCRUD 契约:创建/更新/删除 SQL 形状与冲突映射。
func TestPGStore_CategoryCRUD(t *testing.T) {
	mock := newMock(t)

	// create:唯一冲突 → ErrCategoryTaken
	mock.ExpectQuery(`INSERT INTO cms_categories`).WithArgs("FAQ", "常见问题", 0, true).
		WillReturnError(&pgconn.PgError{Code: "23505"})
	if _, err := NewPGStore(mock).CreateCategory(context.Background(),
		Category{Code: "FAQ", Name: "常见问题", Enabled: true}); !errors.Is(err, ErrCategoryTaken) {
		t.Fatalf("want ErrCategoryTaken, got %v", err)
	}

	// update:在用且改 code → ErrCategoryInUse(预检 code + in-use 各一次)
	mock.ExpectQuery(`SELECT code FROM cms_categories`).WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"code"}).AddRow("NEWS"))
	mock.ExpectQuery(`FROM cms_posts p JOIN cms_categories`).WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"1"}).AddRow(1))
	if err := NewPGStore(mock).UpdateCategory(context.Background(),
		Category{ID: 1, Code: "BLOG", Name: "博客"}); !errors.Is(err, ErrCategoryInUse) {
		t.Fatalf("want ErrCategoryInUse, got %v", err)
	}

	// update:在用但 code 不变(仅停用)放行
	mock.ExpectQuery(`SELECT code FROM cms_categories`).WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"code"}).AddRow("NEWS"))
	mock.ExpectQuery(`FROM cms_posts p JOIN cms_categories`).WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectExec(`UPDATE cms_categories`).WithArgs(int64(1), "NEWS", "动态", 2, false).
		WillReturnResult(pgconn.NewCommandTag("UPDATE 1"))
	if err := NewPGStore(mock).UpdateCategory(context.Background(),
		Category{ID: 1, Code: "NEWS", Name: "动态", SortNo: 2, Enabled: false}); err != nil {
		t.Fatalf("disable in-use: %v", err)
	}

	// delete:在用 → ErrCategoryInUse
	mock.ExpectQuery(`FROM cms_posts p JOIN cms_categories`).WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"1"}).AddRow(1))
	if err := NewPGStore(mock).DeleteCategory(context.Background(), 1); !errors.Is(err, ErrCategoryInUse) {
		t.Fatalf("want ErrCategoryInUse, got %v", err)
	}

	// list:sort_no, code 排序
	mock.ExpectQuery(`ORDER BY sort_no, code`).WillReturnRows(catRow(1, "NEWS", true).AddRow(
		[]interface{}{int64(2), "ARTICLE", "文章", 1, true, "2026-08-28 10:00"}...))
	got, err := NewPGStore(mock).ListCategories(context.Background())
	if err != nil || len(got) != 2 {
		t.Fatalf("got=%v err=%v", got, err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestPGStore_PostCategoryNotUsable 契约:分类不存在或停用时建文被拒。
func TestPGStore_PostCategoryNotUsable(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(`SELECT enabled FROM cms_categories`).WithArgs("OFF").
		WillReturnError(pgx.ErrNoRows)
	if _, err := NewPGStore(mock).CreatePost(context.Background(),
		Post{Slug: "x", Title: "t", Category: "OFF", Summary: "s", Content: "c"}); !errors.Is(err, ErrInvalidPost) {
		t.Fatalf("want ErrInvalidPost, got %v", err)
	}
}

// TestAttRefs 契约:仅识别 ](att/N) 目标,去重升序;RewriteAttRefs 全量重写。
func TestAttRefs(t *testing.T) {
	src := "![a](att/12) [b](att/3.md) ![](att/12) [外](https://x.y/z) ![c](att/7)"
	refs := AttRefs(src)
	if len(refs) != 2 || refs[0] != 7 || refs[1] != 12 {
		t.Fatalf("refs=%v", refs)
	}
	out := RewriteAttRefs(src, func(id int64) string { return "/img/" + strconv.FormatInt(id, 10) })
	want := "![a](/img/12) [b](att/3.md) ![](/img/12) [外](https://x.y/z) ![c](/img/7)"
	if out != want {
		t.Fatalf("out=%q", out)
	}
}
