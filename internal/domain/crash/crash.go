package crash

// 客户端崩溃日志域:App 本地留痕后启动补传(迁移 000136 client_crash_logs)。
// 只追加不回改;上传尽力而为,App 侧成功即删本地文件防重传。

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Store 崩溃日志存储。
type Store interface {
	Insert(ctx context.Context, l Log) error
	ListRecent(ctx context.Context, limit int) ([]Log, error)
}

// Log 一条崩溃记录;log 为 App 端格式化文本(线程/时间/堆栈)。
type Log struct {
	ID          int64     `json:"id"`
	SubjectType string    `json:"subjectType"`
	SubjectID   int64     `json:"subjectId"`
	App         string    `json:"app"`
	Log         string    `json:"log"`
	CreatedAt   time.Time `json:"createdAt"`
}

// MaxLogBytes 单条日志上限,防异常客户端拖库。
const MaxLogBytes = 64 * 1024

// Clip 超长截断(保尾部堆栈)。
func Clip(s string) string {
	if len(s) <= MaxLogBytes {
		return s
	}
	return "...(clipped)... " + s[len(s)-MaxLogBytes:]
}

type pgDB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore PostgreSQL 实现。
type PGStore struct{ db pgDB }

// NewPGStore 构造。
func NewPGStore(db pgDB) *PGStore { return &PGStore{db: db} }

// Insert 追加一条。
func (s *PGStore) Insert(ctx context.Context, l Log) error {
	if l.SubjectType == "" {
		l.SubjectType = "worker"
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO client_crash_logs (subject_type, subject_id, app, log)
		VALUES ($1, $2, $3, $4)`,
		l.SubjectType, l.SubjectID, l.App, Clip(l.Log))
	if err != nil {
		return fmt.Errorf("crash: insert: %w", err)
	}
	return nil
}

// ListRecent 最近 limit 条(新→旧,排查用;上限 200)。
func (s *PGStore) ListRecent(ctx context.Context, limit int) ([]Log, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, subject_type, subject_id, app, log, created_at
		FROM client_crash_logs ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("crash: list: %w", err)
	}
	defer rows.Close()
	var out []Log
	for rows.Next() {
		var l Log
		if err := rows.Scan(&l.ID, &l.SubjectType, &l.SubjectID, &l.App, &l.Log, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("crash: scan: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
