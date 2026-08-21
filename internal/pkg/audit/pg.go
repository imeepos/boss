package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbtx 是 PGWriter 依赖的最小数据库接口。
type dbtx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PGWriter 审计同步写入器(直接 INSERT 到按月分区表 audit_logs)。
type PGWriter struct {
	db dbtx
}

// NewPGWriter 构造 PGWriter;db 传 *pgxpool.Pool 或测试 mock。
func NewPGWriter(db dbtx) *PGWriter {
	return &PGWriter{db: db}
}

// Write 写入一条审计;detail 序列化为 JSONB。
func (w *PGWriter) Write(ctx context.Context, e Event) error {
	detail := "{}"
	if e.Detail != nil {
		b, err := json.Marshal(e.Detail)
		if err != nil {
			return fmt.Errorf("audit: marshal detail: %w", err)
		}
		detail = string(b)
	}
	_, err := w.db.Exec(ctx,
		`INSERT INTO audit_logs(account_id, action, target_type, target_id, detail, ip)
		 VALUES($1,$2,$3,$4,$5,$6)`,
		e.AccountID, e.Action, e.TargetType, e.TargetID, detail, nilIfEmpty(e.IP))
	if err != nil {
		return fmt.Errorf("audit: write: %w", err)
	}
	return nil
}

// List 查询审计;按人/时间/类型过滤 + 分页(按 created_at 倒序);联出操作人姓名。
func (w *PGWriter) List(ctx context.Context, q Query) ([]Entry, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 1 << 30
	}
	rows, err := w.db.Query(ctx, `
		SELECT l.id, l.account_id, l.action, l.target_type, COALESCE(l.target_id, ''),
		       l.detail::text, COALESCE(l.ip::text, ''), l.created_at,
		       COALESCE(a.real_name, '')
		FROM audit_logs l
		LEFT JOIN accounts a ON a.id = l.account_id
		WHERE ($1 = 0 OR l.account_id = $1)
		  AND ($2 = '' OR l.action = $2)
		  AND ($3 = '' OR l.target_type = $3)
		ORDER BY l.created_at DESC
		LIMIT $4 OFFSET $5`,
		q.AccountID, q.Action, q.TargetType, limit, q.Offset)
	if err != nil {
		return nil, fmt.Errorf("audit: list: %w", err)
	}
	defer rows.Close()
	out := make([]Entry, 0)
	for rows.Next() {
		var en Entry
		if err := rows.Scan(&en.ID, &en.AccountID, &en.Action, &en.TargetType, &en.TargetID, &en.Detail, &en.IP, &en.CreatedAt, &en.Operator); err != nil {
			return nil, fmt.Errorf("audit: scan: %w", err)
		}
		out = append(out, en)
	}
	return out, rows.Err()
}

// nilIfEmpty 空串归 NULL(inet 列可空)。
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
