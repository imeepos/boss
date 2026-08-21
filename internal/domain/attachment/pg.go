package attachment

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// Get 按 id 查附件(不含已删)。
func (s *PGStore) Get(ctx context.Context, id int64) (*Attachment, error) {
	return scanOne(s.db.QueryRow(ctx, `SELECT `+cols+` FROM attachments WHERE id=$1 AND deleted_at IS NULL`, id))
}

// ListByUploader 列出上传者的附件(新→旧,不含已删)。
func (s *PGStore) ListByUploader(ctx context.Context, uploaderType string, uploaderID int64, limit int) ([]Attachment, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(ctx,
		`SELECT `+cols+` FROM attachments WHERE uploader_type=$1 AND uploader_id=$2 AND deleted_at IS NULL ORDER BY id DESC LIMIT $3`,
		uploaderType, uploaderID, limit)
	if err != nil {
		return nil, fmt.Errorf("attachment: list: %w", err)
	}
	defer rows.Close()
	return scanRows(rows)
}

// List 按筛选查附件(新→旧,不含已删),返回当页 items 与命中总数。
func (s *PGStore) List(ctx context.Context, f ListFilter) ([]Attachment, int, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	where := "deleted_at IS NULL"
	var args []any
	if f.UploaderType != "" && f.UploaderID > 0 {
		args = append(args, f.UploaderType, f.UploaderID)
		where += fmt.Sprintf(" AND uploader_type=$%d AND uploader_id=$%d", len(args)-1, len(args))
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		args = append(args, "%"+kw+"%")
		where += fmt.Sprintf(" AND file_name ILIKE $%d", len(args))
	}
	var total int
	if err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM attachments WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("attachment: count: %w", err)
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := s.db.Query(ctx,
		`SELECT `+cols+` FROM attachments WHERE `+where+
			fmt.Sprintf(" ORDER BY id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)),
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("attachment: list: %w", err)
	}
	defer rows.Close()
	items, err := scanRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Delete 软删除:置 deleted_at;不存在或已删返回 ErrNotFound。
func (s *PGStore) Delete(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE attachments SET deleted_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("attachment: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetByIDs 按 id 批量查(去重去非正,不含已删,id 降序)。
func (s *PGStore) GetByIDs(ctx context.Context, ids []int64) ([]Attachment, error) {
	uniq := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok || id <= 0 {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return []Attachment{}, nil
	}
	rows, err := s.db.Query(ctx,
		`SELECT `+cols+` FROM attachments WHERE id = ANY($1) AND deleted_at IS NULL ORDER BY id DESC`, uniq)
	if err != nil {
		return nil, fmt.Errorf("attachment: get-by-ids: %w", err)
	}
	defer rows.Close()
	return scanRows(rows)
}

// scanRows 迭代扫描行集合为附件列表。
func scanRows(rows pgx.Rows) ([]Attachment, error) {
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
