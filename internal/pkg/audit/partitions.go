package audit

import (
	"context"
	"fmt"
	"time"
)

// EnsurePartitions 预建 audit_logs 按月分区(当月起 ahead 个月),幂等。
// 修复 E14:000001 只建了 2025_08 + default,无此调用则后续月份全部落入
// default 分区,按月查询失去分区裁剪(architecture-review 发现 1.1/1.4)。
func (w *PGWriter) EnsurePartitions(ctx context.Context, now time.Time, ahead int) error {
	if ahead < 1 {
		ahead = 1
	}
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	for i := 0; i <= ahead; i++ {
		end := month.AddDate(0, 1, 0)
		name := fmt.Sprintf("audit_logs_%s", month.Format("2006_01"))
		if _, err := w.db.Exec(ctx, fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS %s PARTITION OF audit_logs
			 FOR VALUES FROM ('%s') TO ('%s')`,
			name, month.Format(time.DateOnly), end.Format(time.DateOnly))); err != nil {
			return fmt.Errorf("audit: ensure partition %s: %w", name, err)
		}
		month = end
	}
	return nil
}
