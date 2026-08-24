package cms

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const timeFmt = "YYYY-MM-DD HH24:MI"

// dbtx 最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

const cols = `id, slug, title, category, summary, COALESCE(cover_attachment_id, 0), content, status,
	COALESCE(TO_CHAR(published_at, '` + timeFmt + `'), ''),
	version, COALESCE(author_name, ''), COALESCE(TO_CHAR(updated_at, '` + timeFmt + `'), '')`

// PGStore Service 的 PostgreSQL 实现。
type PGStore struct{ db dbtx }

func NewPGStore(db dbtx) *PGStore { return &PGStore{db: db} }

func scanPost(rows pgx.Rows, p *Post) error {
	return rows.Scan(&p.ID, &p.Slug, &p.Title, &p.Category, &p.Summary, &p.CoverAttachment,
		&p.Content, &p.Status, &p.PublishedAt, &p.Version, &p.AuthorName, &p.UpdatedAt)
}

func (s *PGStore) ListPosts(ctx context.Context) ([]Post, error) {
	rows, err := s.db.Query(ctx, `SELECT `+cols+` FROM cms_posts ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("cms: list posts: %w", err)
	}
	defer rows.Close()
	out := make([]Post, 0)
	for rows.Next() {
		var p Post
		if err := scanPost(rows, &p); err != nil {
			return nil, fmt.Errorf("cms: scan post: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListPublished 官网匿名读:仅 PUBLISHED,发布时间倒序,limit 上限 50。
func (s *PGStore) ListPublished(ctx context.Context, category string, limit int) ([]Post, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	q := `SELECT ` + cols + ` FROM cms_posts WHERE status='PUBLISHED'`
	args := []any{limit}
	if category != "" {
		q += ` AND category=$2`
		args = append(args, category)
	}
	q += ` ORDER BY published_at DESC LIMIT $1`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("cms: list published: %w", err)
	}
	defer rows.Close()
	out := make([]Post, 0)
	for rows.Next() {
		var p Post
		if err := scanPost(rows, &p); err != nil {
			return nil, fmt.Errorf("cms: scan published: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetPublishedBySlug 官网详情读:非 PUBLISHED 一律 ErrPostNotFound(不泄露草稿存在性)。
func (s *PGStore) GetPublishedBySlug(ctx context.Context, slug string) (*Post, error) {
	var p Post
	err := s.db.QueryRow(ctx,
		`SELECT `+cols+` FROM cms_posts WHERE slug=$1 AND status='PUBLISHED'`, slug).
		Scan(&p.ID, &p.Slug, &p.Title, &p.Category, &p.Summary, &p.CoverAttachment,
			&p.Content, &p.Status, &p.PublishedAt, &p.Version, &p.AuthorName, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("cms: get by slug: %w", err)
	}
	return &p, nil
}

func (s *PGStore) CreatePost(ctx context.Context, p Post) (int64, error) {
	p.normalize()
	if err := p.validate(); err != nil {
		return 0, err
	}
	if err := s.categoryUsable(ctx, p.Category); err != nil {
		return 0, err
	}
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO cms_posts(slug, title, category, summary, cover_attachment_id, content, status, author_name,
			published_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8, CASE WHEN $7='PUBLISHED' THEN now() END) RETURNING id`,
		p.Slug, p.Title, p.Category, p.Summary, intOrNil(p.CoverAttachment), p.Content, p.Status, p.AuthorName).
		Scan(&id)
	if isUniqueViolation(err) {
		return 0, ErrSlugTaken
	}
	if err != nil {
		return 0, fmt.Errorf("cms: create post: %w", err)
	}
	return id, nil
}

func (s *PGStore) UpdatePost(ctx context.Context, p Post) error {
	p.normalize()
	if err := p.validate(); err != nil {
		return err
	}
	if err := s.categoryUsable(ctx, p.Category); err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE cms_posts SET slug=$2, title=$3, category=$4, summary=$5, cover_attachment_id=$6,
			content=$7, status=$8, author_name=$9,
			published_at=CASE WHEN $8='PUBLISHED' AND published_at IS NULL THEN now() ELSE published_at END,
			version=version+1, updated_at=now()
		WHERE id=$1`,
		p.ID, p.Slug, p.Title, p.Category, p.Summary, intOrNil(p.CoverAttachment), p.Content, p.Status, p.AuthorName)
	if isUniqueViolation(err) {
		return ErrSlugTaken
	}
	if err != nil {
		return fmt.Errorf("cms: update post: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPostNotFound
	}
	return nil
}

func (s *PGStore) DeletePost(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM cms_posts WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("cms: delete post: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPostNotFound
	}
	return nil
}

// normalize 空 category/status 落默认值,减少前端必传字段。
func (p *Post) normalize() {
	if p.Category == "" {
		p.Category = "NEWS"
	}
	if p.Status == "" {
		p.Status = StatusDraft
	}
}

func intOrNil(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
