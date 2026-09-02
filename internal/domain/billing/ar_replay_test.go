package billing

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// ReplayEvents fan-trap 回归:重算必须是"每张事实表独立聚合再相减"的标量子查询形态。
// 2026-09 审计发现按 customer_id 三表横向 LEFT JOIN 后 SUM,笛卡尔积使金额按行数倍增。
func TestPGStoreReplayEvents_FanTrapFree(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectQuery(`FROM ar_replay_events`).
		WithArgs("payment").
		WillReturnRows(pool.NewRows([]string{"id", "customer_id", "source_type", "source_id", "event_type", "payload", "created_at"}).
			AddRow(int64(1), int64(5), "payment", int64(10), "PAID", "{}", time.Now()))
	pool.ExpectExec(`(?s)UPDATE arrears a SET amount = .{1,3}SELECT COALESCE.SUM.b.amount.,0.`).
		WithArgs(int64(5)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(pool)
	n, err := s.ReplayEvents(context.Background(), "payment")
	if err != nil {
		t.Fatalf("ReplayEvents: %v", err)
	}
	if n != 1 {
		t.Fatalf("replayed=%d, want 1", n)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
