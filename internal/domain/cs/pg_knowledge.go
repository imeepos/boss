package cs

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrArticleNotFound 文章不存在。
var ErrArticleNotFound = errors.New("cs: article not found")

// dbtx 是 PGKnowledgeStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGKnowledgeStore 是 KnowledgeService 的 PostgreSQL 实现。
type PGKnowledgeStore struct{ db dbtx }

func NewPGKnowledgeStore(db dbtx) *PGKnowledgeStore { return &PGKnowledgeStore{db: db} }

func (s *PGKnowledgeStore) ListArticles(ctx context.Context) ([]KnowledgeArticle, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, code, title, content, status, version, COALESCE(owner_id, 0),
		       COALESCE(TO_CHAR(published_at, 'YYYY-MM-DD HH24:MI'), ''),
		       COALESCE(TO_CHAR(updated_at, 'YYYY-MM-DD HH24:MI'), '')
		FROM cs_knowledge_articles ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("cs: list articles: %w", err)
	}
	defer rows.Close()
	out := make([]KnowledgeArticle, 0)
	for rows.Next() {
		var a KnowledgeArticle
		if err := rows.Scan(&a.ID, &a.Code, &a.Title, &a.Content, &a.Status, &a.Version,
			&a.OwnerID, &a.PublishedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("cs: scan article: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *PGKnowledgeStore) GetArticle(ctx context.Context, id int64) (*KnowledgeArticle, error) {
	var a KnowledgeArticle
	err := s.db.QueryRow(ctx, `
		SELECT id, code, title, content, status, version, COALESCE(owner_id, 0),
		       COALESCE(TO_CHAR(published_at, 'YYYY-MM-DD HH24:MI'), ''),
		       COALESCE(TO_CHAR(updated_at, 'YYYY-MM-DD HH24:MI'), '')
		FROM cs_knowledge_articles WHERE id = $1`, id).
		Scan(&a.ID, &a.Code, &a.Title, &a.Content, &a.Status, &a.Version, &a.OwnerID, &a.PublishedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrArticleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("cs: get article: %w", err)
	}
	return &a, nil
}

func (s *PGKnowledgeStore) CreateArticle(ctx context.Context, a KnowledgeArticle) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO cs_knowledge_articles(code, title, content, status, version, owner_id)
		VALUES($1,$2,$3,$4,1,$5) RETURNING id`,
		a.Code, a.Title, a.Content, a.Status, intOrNil(a.OwnerID)).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("cs: create article: %w", err)
	}
	return id, nil
}

func (s *PGKnowledgeStore) UpdateArticle(ctx context.Context, a KnowledgeArticle) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE cs_knowledge_articles SET code=$2, title=$3, content=$4, status=$5,
			version=version+1, owner_id=$6,
			published_at=CASE WHEN $5='PUBLISHED' AND published_at IS NULL THEN now() ELSE published_at END,
			updated_at=now()
		WHERE id=$1`, a.ID, a.Code, a.Title, a.Content, a.Status, intOrNil(a.OwnerID))
	if err != nil {
		return fmt.Errorf("cs: update article: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrArticleNotFound
	}
	return nil
}

func (s *PGKnowledgeStore) DeleteArticle(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM cs_knowledge_articles WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("cs: delete article: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrArticleNotFound
	}
	return nil
}

func intOrNil(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}
