package attachment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrNotFound 附件登记不存在。
var ErrNotFound = errors.New("attachment: not found")

// dbtx 是 PGStore 依赖的最小数据库接口。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 是 Store 的 PostgreSQL 实现。
type PGStore struct{ db dbtx }

// NewPGStore 构造 PGStore。
func NewPGStore(db dbtx) *PGStore { return &PGStore{db: db} }

const cols = `id, object_key, file_name, content_type, size_bytes, uploader_type, uploader_id, created_at`

func scanOne(row pgx.Row) (*Attachment, error) {
	var at Attachment
	var createdAt pgtype.Timestamptz
	if err := row.Scan(&at.ID, &at.ObjectKey, &at.FileName, &at.ContentType,
		&at.SizeBytes, &at.UploaderType, &at.UploaderID, &createdAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("attachment: scan: %w", err)
	}
	if createdAt.Valid {
		at.CreatedAt = createdAt.Time.Format(time.RFC3339)
	}
	return &at, nil
}

// Create 登记附件元数据,回填自增 id/createdAt。
func (s *PGStore) Create(ctx context.Context, at *Attachment) (*Attachment, error) {
	if !ValidUploaderType(at.UploaderType) {
		return nil, ErrInvalidUploader
	}
	return scanOne(s.db.QueryRow(ctx, `
		INSERT INTO attachments (object_key, file_name, content_type, size_bytes, uploader_type, uploader_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+cols, at.ObjectKey, at.FileName, at.ContentType, at.SizeBytes, at.UploaderType, at.UploaderID))
}

// Get 按 id 查附件。
func (s *PGStore) Get(ctx context.Context, id int64) (*Attachment, error) {
	return scanOne(s.db.QueryRow(ctx, `SELECT `+cols+` FROM attachments WHERE id=$1`, id))
}

// ListByUploader 列出上传者的附件(新→旧)。
func (s *PGStore) ListByUploader(ctx context.Context, uploaderType string, uploaderID int64, limit int) ([]Attachment, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(ctx,
		`SELECT `+cols+` FROM attachments WHERE uploader_type=$1 AND uploader_id=$2 ORDER BY id DESC LIMIT $3`,
		uploaderType, uploaderID, limit)
	if err != nil {
		return nil, fmt.Errorf("attachment: list: %w", err)
	}
	defer rows.Close()
	out := make([]Attachment, 0)
	for rows.Next() {
		var at Attachment
		var createdAt pgtype.Timestamptz
		if err := rows.Scan(&at.ID, &at.ObjectKey, &at.FileName, &at.ContentType,
			&at.SizeBytes, &at.UploaderType, &at.UploaderID, &createdAt); err != nil {
			return nil, fmt.Errorf("attachment: scan: %w", err)
		}
		if createdAt.Valid {
			at.CreatedAt = createdAt.Time.Format(time.RFC3339)
		}
		out = append(out, at)
	}
	return out, rows.Err()
}
