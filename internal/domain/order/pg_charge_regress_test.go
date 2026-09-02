package order

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// 区域月费覆盖 join 回归:连接条件必须是 ro.region_path = o.region_path。
// 2026-09 审计发现同列自比(o.region_path = o.region_path)恒真,区域覆盖永不生效;
// pgxmock 以正则锁定正确形态,旧 SQL 不含 ro. 前缀即失败。
func TestPrepaidMonthlyFee_RegionJoin(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`ro\.region_path = o\.region_path`).
		WithArgs(int64(7)).
		WillReturnRows(pgxmock.NewRows([]string{"monthly_fee"}).AddRow(88.5))

	s := NewPGStore(mock, stubExists{})
	got, err := s.prepaidMonthlyFee(context.Background(), 7)
	if err != nil {
		t.Fatalf("prepaidMonthlyFee: %v", err)
	}
	if got != 88.5 {
		t.Fatalf("fee=%v, want 88.5", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
