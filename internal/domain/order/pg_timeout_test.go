package order

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// ReleaseExpiredReserves(预占超时释放)形状回归:候选 SELECT + 逐单
// 端口回收/流水 RELEASED/status release 迁移(RESERVED→PENDING)。
func TestPGStore_ReleaseExpiredReserves(t *testing.T) {
	cutoff := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	t.Run("超时候选逐单释放", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(11)).AddRow(int64(12))
		mock.ExpectQuery(`SELECT o.id FROM orders o`).
			WithArgs(cutoff).
			WillReturnRows(rows)
		for _, id := range []int64{11, 12} {
			mock.ExpectExec(`UPDATE ports SET status = 'IDLE', order_id = NULL`).
				WithArgs(id).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			mock.ExpectExec(`UPDATE reserve_records SET status = 'RELEASED'`).
				WithArgs(id).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			// transitionStatus: SELECT status → UPDATE status
			mock.ExpectQuery(`SELECT status FROM orders WHERE id`).
				WithArgs(id).WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("RESERVED"))
			mock.ExpectExec(`UPDATE orders SET status`).
				WithArgs(id, "PENDING").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		}

		s := NewPGStore(mock, nil)
		ids, err := s.ReleaseExpiredReserves(context.Background(), cutoff)
		if err != nil {
			t.Fatalf("ReleaseExpiredReserves: %v", err)
		}
		if len(ids) != 2 || ids[0] != 11 || ids[1] != 12 {
			t.Fatalf("ids=%v", ids)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("无候选空转", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT o.id FROM orders o`).
			WithArgs(cutoff).
			WillReturnRows(pgxmock.NewRows([]string{"id"}))

		s := NewPGStore(mock, nil)
		ids, err := s.ReleaseExpiredReserves(context.Background(), cutoff)
		if err != nil {
			t.Fatalf("ReleaseExpiredReserves: %v", err)
		}
		if len(ids) != 0 {
			t.Fatalf("ids=%v, want empty", ids)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}
