package audit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// EnsurePartitions 预建 audit_logs 按月分区(当月起 ahead 个月),幂等。
// 修复 E14:000001 只建了 2025_08 + default,无此调用则后续月份全部落入
// default 分区,按月查询失去分区裁剪(architecture-review 发现 1.1/1.4)。
// 修复 0001-postmortem:default 分区已积压目标月数据时(PG 23514 拒建),
// 自动搬移后重试,避免启动崩溃循环。
func (w *PGWriter) EnsurePartitions(ctx context.Context, now time.Time, ahead int) error {
	if ahead < 1 {
		ahead = 1
	}
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	for i := 0; i <= ahead; i++ {
		end := month.AddDate(0, 1, 0)
		name := fmt.Sprintf("audit_logs_%s", month.Format("2006_01"))
		if err := w.ensureMonthPartition(ctx, name, month, end); err != nil {
			return fmt.Errorf("audit: ensure partition %s: %w", name, err)
		}
		month = end
	}
	return nil
}

// ensureMonthPartition 建单月分区;若 default 分区已有该月数据被 PG 拒绝
// (SQLSTATE 23514),先在事务内把数据暂存搬出、建分区、放回,全程原子。
func (w *PGWriter) ensureMonthPartition(ctx context.Context, name string, from, to time.Time) error {
	createSQL := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s PARTITION OF audit_logs
		 FOR VALUES FROM ('%s') TO ('%s')`,
		name, from.Format(time.DateOnly), to.Format(time.DateOnly))
	_, err := w.db.Exec(ctx, createSQL)
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		return err
	}
	return w.migrateDefaultRows(ctx, name, from, to, createSQL)
}

// migrateDefaultRows default 分区 [from,to) 数据搬到新分区,失败整体回滚。
func (w *PGWriter) migrateDefaultRows(ctx context.Context, name string, from, to time.Time, createSQL string) error {
	tx, err := w.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	stage := name + "_stage"
	lo, hi := from.Format(time.DateOnly), to.Format(time.DateOnly)
	stmts := []string{
		fmt.Sprintf(`CREATE TABLE %s (LIKE audit_logs INCLUDING DEFAULTS)`, stage),
		fmt.Sprintf(`INSERT INTO %s SELECT * FROM audit_logs_default WHERE created_at >= '%s' AND created_at < '%s'`, stage, lo, hi),
		fmt.Sprintf(`DELETE FROM audit_logs_default WHERE created_at >= '%s' AND created_at < '%s'`, lo, hi),
		createSQL,
		fmt.Sprintf(`INSERT INTO audit_logs SELECT * FROM %s`, stage),
		fmt.Sprintf(`DROP TABLE %s`, stage),
	}
	for _, s := range stmts {
		if _, err := tx.Exec(ctx, s); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
