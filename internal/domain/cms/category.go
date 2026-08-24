// 分类字典:cms_categories,code 全局唯一(大写蛇形),name 展示名。
// posts.category 软引用 code;启用态只影响新文章可选,已发布文章不受影响。
package cms

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"
)

var (
	// ErrCategoryNotFound 分类不存在(或已删)。
	ErrCategoryNotFound = errors.New("cms: category not found")
	// ErrCategoryTaken code 已被占用。
	ErrCategoryTaken = errors.New("cms: category code already taken")
	// ErrCategoryInUse 分类仍被文章引用,禁改 code/禁删。
	ErrCategoryInUse = errors.New("cms: category in use")
	// ErrInvalidCategory 字段校验失败(code/name 长度或格式)。
	ErrInvalidCategory = errors.New("cms: invalid category")
)

// codeRe 大写字母数字蛇形,2~32 位(与迁移 VARCHAR(32) 对齐)。
var codeRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,31}$`)

// Category cms_categories 行投影。
type Category struct {
	ID        int64  `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortNo    int    `json:"sortNo"`
	Enabled   bool   `json:"enabled"`
	UpdatedAt string `json:"updatedAt"`
}

func (c *Category) validate() error {
	if !codeRe.MatchString(c.Code) {
		return ErrInvalidCategory
	}
	if c.Name == "" || len(c.Name) > 64 {
		return ErrInvalidCategory
	}
	return nil
}

// CategoryStore 分类管理面接口。
type CategoryStore interface {
	ListCategories(ctx context.Context) ([]Category, error)
	CreateCategory(ctx context.Context, c Category) (int64, error)
	UpdateCategory(ctx context.Context, c Category) error
	DeleteCategory(ctx context.Context, id int64) error
}

const catCols = `id, code, name, sort_no, enabled, TO_CHAR(updated_at, '` + timeFmt + `')`

func scanCategory(rows pgx.Rows, c *Category) error {
	return rows.Scan(&c.ID, &c.Code, &c.Name, &c.SortNo, &c.Enabled, &c.UpdatedAt)
}

// ListCategories 全量字典,sort_no 升序再 code 升序;给下拉与字典页共用。
func (s *PGStore) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.db.Query(ctx, `SELECT `+catCols+` FROM cms_categories ORDER BY sort_no, code`)
	if err != nil {
		return nil, fmt.Errorf("cms: list categories: %w", err)
	}
	defer rows.Close()
	out := make([]Category, 0)
	for rows.Next() {
		var c Category
		if err := scanCategory(rows, &c); err != nil {
			return nil, fmt.Errorf("cms: scan category: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *PGStore) CreateCategory(ctx context.Context, c Category) (int64, error) {
	if err := c.validate(); err != nil {
		return 0, err
	}
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO cms_categories(code, name, sort_no, enabled) VALUES($1,$2,$3,$4) RETURNING id`,
		c.Code, c.Name, c.SortNo, c.Enabled).Scan(&id)
	if isUniqueViolation(err) {
		return 0, ErrCategoryTaken
	}
	if err != nil {
		return 0, fmt.Errorf("cms: create category: %w", err)
	}
	return id, nil
}

// UpdateCategory code 仍被文章引用时禁改(防 FK CASCADE 静默改写存量文章归属);
// 停用允许,存量文章不受影响,仅新文章不可再选。
func (s *PGStore) UpdateCategory(ctx context.Context, c Category) error {
	if err := c.validate(); err != nil {
		return err
	}
	var curCode string
	err := s.db.QueryRow(ctx, `SELECT code FROM cms_categories WHERE id=$1`, c.ID).Scan(&curCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCategoryNotFound
	}
	if err != nil {
		return fmt.Errorf("cms: get category code: %w", err)
	}
	inUse, err := s.categoryInUse(ctx, c.ID)
	if err != nil {
		return err
	}
	if inUse && c.Code != curCode {
		return ErrCategoryInUse
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE cms_categories SET code=$2, name=$3, sort_no=$4, enabled=$5, updated_at=now()
		WHERE id=$1`,
		c.ID, c.Code, c.Name, c.SortNo, c.Enabled)
	if isUniqueViolation(err) {
		return ErrCategoryTaken
	}
	if err != nil {
		return fmt.Errorf("cms: update category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

// DeleteCategory 仍被文章引用时拒绝(ErrCategoryInUse),防存量文章 category 悬空。
func (s *PGStore) DeleteCategory(ctx context.Context, id int64) error {
	inUse, err := s.categoryInUse(ctx, id)
	if err != nil {
		return err
	}
	if inUse {
		return ErrCategoryInUse
	}
	tag, err := s.db.Exec(ctx, `DELETE FROM cms_categories WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("cms: delete category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

func (s *PGStore) categoryInUse(ctx context.Context, id int64) (bool, error) {
	var n int
	err := s.db.QueryRow(ctx,
		`SELECT 1 FROM cms_posts p JOIN cms_categories c ON p.category=c.code
		WHERE c.id=$1 LIMIT 1`, id).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cms: category in use: %w", err)
	}
	return true, nil
}

// categoryUsable 落库前校验文章分类存在且启用(新文章不可选停用分类)。
func (s *PGStore) categoryUsable(ctx context.Context, code string) error {
	var enabled bool
	err := s.db.QueryRow(ctx,
		`SELECT enabled FROM cms_categories WHERE code=$1`, code).Scan(&enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidPost
	}
	if err != nil {
		return fmt.Errorf("cms: category usable: %w", err)
	}
	if !enabled {
		return ErrInvalidPost
	}
	return nil
}
