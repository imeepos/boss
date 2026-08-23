package loy

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestTierOf_NoLevel(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	s := NewPGStore(mock, nil, nil)

	mock.ExpectQuery(`SELECT COALESCE\(SUM\(delta\),0\)`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"earned"}).AddRow(int64(500)))
	mock.ExpectQuery(`SELECT level_id, name, min_points, status FROM loy_levels`).
		WithArgs(int64(500)).
		WillReturnRows(mock.NewRows([]string{"level_id", "name", "min_points", "status"}))

	lv, err := s.TierOf(context.Background(), 7)
	if err != nil || lv != nil {
		t.Fatalf("lv=%v err=%v", lv, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestEarnForPayment_NoRule(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	s := NewPGStore(mock, nil, nil)

	mock.ExpectQuery(`FROM loy_earn_rules`).
		WillReturnRows(mock.NewRows([]string{"rule_id", "points_per_yuan", "min_cents", "expire_days", "status"}))

	pts, err := s.EarnForPayment(context.Background(), 11, 7, 10000)
	if err != nil || pts != 0 {
		t.Fatalf("pts=%d err=%v", pts, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestEarnForPayment_BelowMin(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	s := NewPGStore(mock, nil, nil)

	mock.ExpectQuery(`FROM loy_earn_rules`).
		WillReturnRows(mock.NewRows(
			[]string{"rule_id", "points_per_yuan", "min_cents", "expire_days", "status"}).
			AddRow(int64(1), 2, 5000, 0, "ENABLED"))

	pts, err := s.EarnForPayment(context.Background(), 11, 7, 4900)
	if err != nil || pts != 0 {
		t.Fatalf("pts=%d err=%v", pts, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestRollbackPayment_NeverEarned(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	s := NewPGStore(mock, nil, nil)

	mock.ExpectQuery(`SELECT delta FROM loy_point_entries`).
		WithArgs(ReasonPayEarn, int64(11), int64(7)).
		WillReturnRows(mock.NewRows([]string{"delta"}))

	pts, err := s.RollbackPayment(context.Background(), 11, 7)
	if err != nil || pts != 0 {
		t.Fatalf("pts=%d err=%v", pts, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestRollbackPayment_AlreadyRolled(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	s := NewPGStore(mock, nil, nil)

	mock.ExpectQuery(`SELECT delta FROM loy_point_entries`).
		WithArgs(ReasonPayEarn, int64(11), int64(7)).
		WillReturnRows(mock.NewRows([]string{"delta"}).AddRow(int64(200)))
	mock.ExpectQuery(`SELECT 1 FROM loy_point_entries`).
		WithArgs(ReasonPayRoll, int64(11)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(1))

	pts, err := s.RollbackPayment(context.Background(), 11, 7)
	if err != nil || pts != 0 {
		t.Fatalf("pts=%d err=%v", pts, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestExpireDue_NothingDue(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	s := NewPGStore(mock, nil, nil)

	mock.ExpectQuery(`GROUP BY e\.customer_id`).
		WillReturnRows(mock.NewRows([]string{"customer_id", "sum", "min"}))

	n, err := s.ExpireDue(context.Background())
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
