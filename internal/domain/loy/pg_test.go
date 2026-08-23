package loy

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestAdjust_Mock(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	s := NewPGStore(mock, nil, nil)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO loy_point_ledgers`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"customer_id"}))
	mock.ExpectQuery(`UPDATE loy_point_ledgers`).
		WithArgs(int64(7), int64(500)).
		WillReturnRows(mock.NewRows([]string{"balance"}).AddRow(int64(500)))
	mock.ExpectExec(`INSERT INTO loy_point_entries`).
		WithArgs(int64(7), int64(500), int64(500), ReasonAdjust, int64(0), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	bal, err := s.Adjust(context.Background(), 7, 500, ReasonAdjust)
	if err != nil || bal != 500 {
		t.Fatalf("bal=%d err=%v", bal, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestExchange_Mock(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	price := func(context.Context, int64) (int64, error) { return 300, nil }
	issued := false
	issue := func(context.Context, int64, int64) (string, error) {
		issued = true
		return "CPN-loy", nil
	}
	s := NewPGStore(mock, price, issue)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO loy_point_ledgers`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"customer_id"}))
	mock.ExpectQuery(`UPDATE loy_point_ledgers`).
		WithArgs(int64(7), int64(-300)).
		WillReturnRows(mock.NewRows([]string{"balance"}).AddRow(int64(200)))
	mock.ExpectExec(`INSERT INTO loy_point_entries`).
		WithArgs(int64(7), int64(-300), int64(200), ReasonExchange, int64(5), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	couponID, cost, err := s.Exchange(context.Background(), 7, 5)
	if err != nil || cost != 300 || couponID != "CPN-loy" || !issued {
		t.Fatalf("coupon=%s cost=%d err=%v issued=%v", couponID, cost, err, issued)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestExchange_IssueFailCompensates(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	price := func(context.Context, int64) (int64, error) { return 300, nil }
	issue := func(context.Context, int64, int64) (string, error) {
		return "", errors.New("issue boom")
	}
	s := NewPGStore(mock, price, issue)

	// 扣积分事务
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO loy_point_ledgers`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"customer_id"}))
	mock.ExpectQuery(`UPDATE loy_point_ledgers`).
		WithArgs(int64(7), int64(-300)).
		WillReturnRows(mock.NewRows([]string{"balance"}).AddRow(int64(0)))
	mock.ExpectExec(`INSERT INTO loy_point_entries`).
		WithArgs(int64(7), int64(-300), int64(0), ReasonExchange, int64(5), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()
	// 补偿回补事务
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO loy_point_ledgers`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"customer_id"}))
	mock.ExpectQuery(`UPDATE loy_point_ledgers`).
		WithArgs(int64(7), int64(300)).
		WillReturnRows(mock.NewRows([]string{"balance"}).AddRow(int64(300)))
	mock.ExpectExec(`INSERT INTO loy_point_entries`).
		WithArgs(int64(7), int64(300), int64(300), ReasonReversal, int64(0), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	if _, _, err := s.Exchange(context.Background(), 7, 5); err == nil {
		t.Fatal("exchange should fail when issue fails")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
