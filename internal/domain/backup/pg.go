// backup 域 PostgreSQL 实现(任务簿记 + 候选表清单)。
package backup

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound 任务不存在。
var ErrNotFound = errors.New("backup: not found")

// ErrBusy 已有任务在执行(进程内串行),稍后重试。
var ErrBusy = errors.New("backup: job already running")

// 系统表不入候选集(自引用/迁移簿记,备份恢复均无意义)。
var excludedTables = map[string]bool{
	"backup_jobs":       true,
	"schema_migrations": true,
}

// PGStore 任务簿记存储。
type PGStore struct{ db *pgxpool.Pool }

// NewPGStore 构造 PGStore。
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{db: pool} }

// ListTables 候选表清单(public 普通表,按名排序,排除系统表)。
func (s *PGStore) ListTables(ctx context.Context) ([]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name`)
	if err != nil {
		return nil, fmt.Errorf("backup: list tables: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0, 64)
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("backup: scan table: %w", err)
		}
		if !excludedTables[t] {
			out = append(out, t)
		}
	}
	return out, rows.Err()
}

// CreateJob 落一条 running 任务,返回 id。
func (s *PGStore) CreateJob(ctx context.Context, kind, scope string, tables []string, operatorID int64) (int64, error) {
	if tables == nil {
		tables = []string{}
	}
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO backup_jobs(kind, scope, tables, status, operator_id)
		VALUES($1, $2, $3, $4, $5) RETURNING id`,
		kind, scope, tables, StatusRunning, operatorID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("backup: create job: %w", err)
	}
	return id, nil
}

const jobCols = `j.id, j.kind, j.scope, j.tables, j.status, j.file_name, j.size_bytes,
	j.table_count, j.row_count, j.error, COALESCE(a.real_name, ''), j.created_at, j.finished_at`

func scanJob(row pgx.Row) (*Job, error) {
	var j Job
	var created, finished pgtype.Timestamptz
	err := row.Scan(&j.ID, &j.Kind, &j.Scope, &j.Tables, &j.Status, &j.FileName, &j.SizeBytes,
		&j.TableCount, &j.RowCount, &j.Error, &j.Operator, &created, &finished)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("backup: scan job: %w", err)
	}
	j.CreatedAt = created.Time.Format(time.RFC3339)
	if finished.Valid {
		j.FinishedAt = finished.Time.Format(time.RFC3339)
	}
	return &j, nil
}

// GetJob 任务详情。
func (s *PGStore) GetJob(ctx context.Context, id int64) (*Job, error) {
	return scanJob(s.db.QueryRow(ctx, `
		SELECT `+jobCols+` FROM backup_jobs j
		LEFT JOIN accounts a ON a.id = j.operator_id WHERE j.id = $1`, id))
}

// ListJobs 任务清单(时间倒序 + kind/status 过滤 + 总数)。
func (s *PGStore) ListJobs(ctx context.Context, f ListFilter) ([]Job, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	if f.Kind != "" {
		args = append(args, f.Kind)
		where += fmt.Sprintf(" AND j.kind = $%d", len(args))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(" AND j.status = $%d", len(args))
	}
	var total int
	if err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM backup_jobs j `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("backup: count jobs: %w", err)
	}
	limit, offset := f.Limit, f.Offset
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.Query(ctx, `
		SELECT `+jobCols+` FROM backup_jobs j
		LEFT JOIN accounts a ON a.id = j.operator_id
		`+where+` ORDER BY j.created_at DESC LIMIT `+fmt.Sprint(limit)+` OFFSET `+fmt.Sprint(offset), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("backup: list jobs: %w", err)
	}
	defer rows.Close()
	out := make([]Job, 0)
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *j)
	}
	return out, total, rows.Err()
}

// FinishJob 收尾:status + 统计 + 错误信息 + 文件名。
func (s *PGStore) FinishJob(ctx context.Context, id int64, status, fileName string, sizeBytes int64, tableCount int, rowCount int64, errMsg string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE backup_jobs
		SET status = $2, file_name = $3, size_bytes = $4, table_count = $5,
		    row_count = $6, error = $7, finished_at = now()
		WHERE id = $1`,
		id, status, fileName, sizeBytes, tableCount, rowCount, errMsg)
	if err != nil {
		return fmt.Errorf("backup: finish job: %w", err)
	}
	return nil
}

// SetFile 补写文件名(恢复任务建行时归档文件已就位)。
func (s *PGStore) SetFile(ctx context.Context, id int64, fileName string) error {
	_, err := s.db.Exec(ctx, `UPDATE backup_jobs SET file_name = $2 WHERE id = $1`, id, fileName)
	if err != nil {
		return fmt.Errorf("backup: set file: %w", err)
	}
	return nil
}

// DeleteJob 删除任务行,返回其文件名(空 = 无文件,调用方自行清理磁盘)。
func (s *PGStore) DeleteJob(ctx context.Context, id int64) (string, error) {
	var fileName string
	err := s.db.QueryRow(ctx,
		`DELETE FROM backup_jobs WHERE id = $1 RETURNING file_name`, id).Scan(&fileName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("backup: delete job: %w", err)
	}
	return fileName, nil
}
