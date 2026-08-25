// 导入任务记录(SYS 域,P2):数据导入中心的历史清单,写入口在各 import handler 成功后。
package user

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ImportTask 一次导入任务的记录行。
type ImportTask struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	Operator  string `json:"operator"` // 操作人姓名(联查快照)
	Total     int    `json:"total"`
	Imported  int    `json:"imported"`
	Failed    int    `json:"failed"`
	Skipped   int    `json:"skipped"`
	Detail    string `json:"detail,omitempty"`
	CreatedAt string `json:"createdAt"`
}

// RecordImportTask 落一条导入记录;total 为空时由成功/失败/跳过数推导;detail 可为 nil。
func (s *PGStore) RecordImportTask(ctx context.Context, kind string, operatorID int64, total, imported, failed, skipped int, detail map[string]any) error {
	if total == 0 {
		total = imported + failed + skipped
	}
	d := "{}"
	if detail != nil {
		b, err := json.Marshal(detail)
		if err != nil {
			return ErrInvalidInput
		}
		d = string(b)
	}
	if _, err := s.db.Exec(ctx,
		`INSERT INTO import_tasks(kind, operator_id, total, imported, failed, skipped, detail) VALUES($1,$2,$3,$4,$5,$6,$7::jsonb)`,
		kind, operatorID, total, imported, failed, skipped, d); err != nil {
		return fmt.Errorf("user: record import task: %w", err)
	}
	return nil
}

// ListImportTasks 导入记录清单(近 200 条,时间倒序)。
func (s *PGStore) ListImportTasks(ctx context.Context) ([]ImportTask, error) {
	rows, err := s.db.Query(ctx, `
		SELECT t.id, t.kind, COALESCE(a.real_name, ''), t.total, t.imported, t.failed, t.skipped,
		       COALESCE(t.detail::text, '{}'), t.created_at
		FROM import_tasks t
		LEFT JOIN accounts a ON a.id = t.operator_id
		ORDER BY t.created_at DESC
		LIMIT 200`)
	if err != nil {
		return nil, fmt.Errorf("user: list import tasks: %w", err)
	}
	defer rows.Close()
	out := make([]ImportTask, 0)
	for rows.Next() {
		var it ImportTask
		var ts pgtype.Timestamptz
		if err := rows.Scan(&it.ID, &it.Kind, &it.Operator, &it.Total, &it.Imported, &it.Failed, &it.Skipped, &it.Detail, &ts); err != nil {
			return nil, fmt.Errorf("user: scan import task: %w", err)
		}
		if ts.Valid {
			it.CreatedAt = ts.Time.Format(time.RFC3339)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
