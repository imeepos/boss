package audit

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGWriter_EnsurePartitions 契约:当月起 ahead+1 个月,月度 RANGE 分区,幂等 IF NOT EXISTS。
func TestPGWriter_EnsurePartitions(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	for _, m := range []struct{ name, from, to string }{
		{"audit_logs_2026_08", "2026-08-01", "2026-09-01"},
		{"audit_logs_2026_09", "2026-09-01", "2026-10-01"},
		{"audit_logs_2026_10", "2026-10-01", "2026-11-01"},
	} {
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS " + m.name + " PARTITION OF audit_logs").
			WillReturnResult(pgxmock.NewResult("CREATE TABLE", 1))
		_ = m // name 已用于 ExpectExec 正则
	}

	w := NewPGWriter(mock)
	if err := w.EnsurePartitions(context.Background(), now, 2); err != nil {
		t.Fatalf("EnsurePartitions: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPGWriter_EnsurePartitions_MinAhead ahead<1 至少预建当月+1。
func TestPGWriter_EnsurePartitions_MinAhead(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	now := time.Date(2026, 12, 31, 23, 0, 0, 0, time.UTC)
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs_2026_12 PARTITION OF audit_logs").
		WillReturnResult(pgxmock.NewResult("CREATE TABLE", 1))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs_2027_01 PARTITION OF audit_logs").
		WillReturnResult(pgxmock.NewResult("CREATE TABLE", 1))

	w := NewPGWriter(mock)
	if err := w.EnsurePartitions(context.Background(), now, 0); err != nil {
		t.Fatalf("EnsurePartitions: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
