package report

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// 巡检 SQL 形状回归:count + array_agg 样本两列扫描;fake Store 走 ErrNoPatrol 分支。
func TestPatrolOrphans_Shape(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	for range orphanChecks {
		mock.ExpectQuery(`SELECT count\(\*\), COALESCE\(\(array_agg`).
			WillReturnRows(pgxmock.NewRows([]string{"count", "ids"}).
				AddRow(int64(0), []int64{}))
	}

	s := NewPGStore(mock)
	got, err := s.PatrolOrphans(context.Background())
	if err != nil {
		t.Fatalf("PatrolOrphans: %v", err)
	}
	if len(got) != len(orphanChecks) {
		t.Fatalf("findings=%d, want %d", len(got), len(orphanChecks))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestReportService_PatrolUnsupported(t *testing.T) {
	r := &ReportService{}
	if _, err := r.PatrolOrphans(context.Background()); err != ErrNoPatrol {
		t.Fatalf("err=%v, want ErrNoPatrol", err)
	}
}
