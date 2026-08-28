package worker

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// 回归:LatestLocationForOrder JOIN dispatch_tickets 后列必须 l. 限定,
// 裸列名在 PG 解析期报 42702 歧义(主链路验收 2026-08-28 捕获)。
func TestLatestLocationForOrderQualifiedSQL(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	reportedAt := time.Date(2026, 8, 28, 6, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT l\.id, l\.worker_id, l\.lat, l\.lng, l\.accuracy_m, l\.speed_mps, l\.bearing, l\.reported_at`).
		WithArgs(int64(594)).
		WillReturnRows(mock.NewRows([]string{"id", "worker_id", "lat", "lng", "accuracy_m", "speed_mps", "bearing", "reported_at"}).
			AddRow(int64(1), int64(6), 14.599, 120.982, 5.0, 1.2, 90.0, reportedAt))

	s := NewPGStore(mock)
	loc, err := s.LatestLocationForOrder(context.Background(), 594)
	if err != nil {
		t.Fatalf("LatestLocationForOrder: %v", err)
	}
	if loc == nil || loc.WorkerID != 6 || loc.Lat != 14.599 {
		t.Fatalf("loc=%+v", loc)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
