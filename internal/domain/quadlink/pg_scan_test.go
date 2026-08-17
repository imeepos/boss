package quadlink

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// W5 写侧(TDD 先行):扫码绑定校验(不一致拒)、拆机必扫码、四码对账。

func TestPGStore_VerifyScan(t *testing.T) {
	t.Run("EPC 与预绑定资产一致 → MATCH 且置 LINKED", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()

		mock.ExpectQuery(`SELECT customer_id FROM orders WHERE id = \$1`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(3)))
		mock.ExpectQuery(`FROM ports`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(11)))
		mock.ExpectQuery(`FROM quad_links`).
			WithArgs(int64(11)).
			WillReturnRows(mock.NewRows(cols).
				AddRow(int64(1), int64(5), int64(3), int64(11), int64(21), int64(1), "主品牌·企业", "UNLINKED"))
		mock.ExpectQuery(`FROM tags`).
			WithArgs("EPC-OK").
			WillReturnRows(mock.NewRows([]string{"id", "bound_asset_id"}).AddRow(int64(9), int64(5)))
		mock.ExpectExec(`UPDATE quad_links SET status = 'LINKED'`).
			WithArgs(int64(1)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectQuery(`INSERT INTO scan_logs`).
			WithArgs(int64(7), int64(2), "张师傅", int64(9), "MATCH").
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(100)))

		s := NewPGStore(mock)
		res, err := s.VerifyScan(context.Background(), ScanReq{
			OrderID: 7, WorkerID: 2, WorkerName: "张师傅", ScannedEPC: "EPC-OK",
		})
		if err != nil {
			t.Fatalf("VerifyScan: %v", err)
		}
		if res != "MATCH" {
			t.Fatalf("res=%s, want MATCH", res)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("EPC 不一致 → ErrScanMismatch 且日志 MISMATCH", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()

		mock.ExpectQuery(`SELECT customer_id FROM orders WHERE id = \$1`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(3)))
		mock.ExpectQuery(`FROM ports`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(11)))
		mock.ExpectQuery(`FROM quad_links`).
			WithArgs(int64(11)).
			WillReturnRows(mock.NewRows(cols).
				AddRow(int64(1), int64(5), int64(3), int64(11), int64(21), int64(1), "主品牌·企业", "UNLINKED"))
		mock.ExpectQuery(`FROM tags`).
			WithArgs("EPC-BAD").
			WillReturnRows(mock.NewRows([]string{"id", "bound_asset_id"}).AddRow(int64(9), int64(99)))
		mock.ExpectQuery(`INSERT INTO scan_logs`).
			WithArgs(int64(7), int64(2), "张师傅", int64(9), "MISMATCH").
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(101)))

		s := NewPGStore(mock)
		_, err := s.VerifyScan(context.Background(), ScanReq{
			OrderID: 7, WorkerID: 2, WorkerName: "张师傅", ScannedEPC: "EPC-BAD",
		})
		if !errors.Is(err, ErrScanMismatch) {
			t.Fatalf("err=%v, want ErrScanMismatch", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("订单未预绑定 → ErrNotPrebound", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()

		mock.ExpectQuery(`SELECT customer_id FROM orders WHERE id = \$1`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(3)))
		mock.ExpectQuery(`FROM ports`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(11)))
		mock.ExpectQuery(`FROM quad_links`).
			WithArgs(int64(11)).
			WillReturnRows(mock.NewRows(cols)) // 无预绑定行

		s := NewPGStore(mock)
		_, err := s.VerifyScan(context.Background(), ScanReq{OrderID: 7, ScannedEPC: "EPC-OK"})
		if !errors.Is(err, ErrNotPrebound) {
			t.Fatalf("err=%v, want ErrNotPrebound", err)
		}
	})
}

func TestPGStore_UnbindRequireScan(t *testing.T) {
	t.Run("扫码一致 → 置 UNLINKED", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()

		mock.ExpectQuery(`SELECT customer_id FROM orders WHERE id = \$1`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(3)))
		mock.ExpectQuery(`FROM quad_links`).
			WithArgs(int64(3)).
			WillReturnRows(mock.NewRows(cols).
				AddRow(int64(1), int64(5), int64(3), int64(11), int64(21), int64(1), "主品牌·企业", "LINKED"))
		mock.ExpectQuery(`FROM tags`).
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"epc_code"}).AddRow("EPC-OK"))
		mock.ExpectExec(`UPDATE quad_links SET status = 'UNLINKED'`).
			WithArgs(int64(1)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		s := NewPGStore(mock)
		if err := s.UnbindRequireScan(context.Background(), 7, "EPC-OK"); err != nil {
			t.Fatalf("UnbindRequireScan: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("码不一致 → 拒", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()

		mock.ExpectQuery(`SELECT customer_id FROM orders WHERE id = \$1`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(3)))
		mock.ExpectQuery(`FROM quad_links`).
			WithArgs(int64(3)).
			WillReturnRows(mock.NewRows(cols).
				AddRow(int64(1), int64(5), int64(3), int64(11), int64(21), int64(1), "主品牌·企业", "LINKED"))
		mock.ExpectQuery(`FROM tags`).
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"epc_code"}).AddRow("EPC-OK"))

		s := NewPGStore(mock)
		if err := s.UnbindRequireScan(context.Background(), 7, "EPC-BAD"); !errors.Is(err, ErrScanMismatch) {
			t.Fatalf("err=%v, want ErrScanMismatch", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("不扫码 → 直接拒(拆机必扫码)", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		s := NewPGStore(mock)
		if err := s.UnbindRequireScan(context.Background(), 7, ""); !errors.Is(err, ErrScanRequired) {
			t.Fatalf("err=%v, want ErrScanRequired", err)
		}
	})
}

func TestPGStore_Reconcile(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()

	// 对账:成员缺失的关联置 CONFLICT。
	mock.ExpectExec(`UPDATE quad_links ql SET status = 'CONFLICT'`).
		WillReturnResult(pgxmock.NewResult("UPDATE", 2))
	// 统计各状态数。
	mock.ExpectQuery(`SELECT status, count(.*) FROM quad_links GROUP BY status`).
		WillReturnRows(mock.NewRows([]string{"status", "count"}).
			AddRow("LINKED", int64(10)).
			AddRow("CONFLICT", int64(2)))

	s := NewPGStore(mock)
	rep, err := s.Reconcile(context.Background())
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if rep.Total != 12 || rep.Linked != 10 || rep.Conflict != 2 {
		t.Fatalf("rep=%+v", rep)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
