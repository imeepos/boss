package asset

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// diffCloseMocks 关单路径公共桩:MISSING 回填 + 未处置差异计数。
func diffCloseMocks(mock pgxmock.PgxPoolIface, taskID int64, open int) {
	mock.ExpectExec(`UPDATE stocktake_items SET kind = 'MISSING'`).
		WithArgs(taskID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectQuery(`SELECT count\(\*\) FROM stocktake_items`).
		WithArgs(taskID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(open))
}

// HandleStocktakeDiff 关单(TDD 先行):全处置完置 DONE,存在未处置差异拒绝。
func TestPGStore_HandleStocktakeDiff(t *testing.T) {
	t.Run("无未处置差异 DOING→DONE", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		diffCloseMocks(mock, 5, 0)
		mock.ExpectExec(`UPDATE stocktakes SET status = 'DONE', progress = 100`).
			WithArgs(int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.HandleStocktakeDiff(context.Background(), 5); err != nil {
			t.Fatalf("HandleStocktakeDiff: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("存在未处置差异拒绝关单", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		diffCloseMocks(mock, 5, 2)
		s := NewPGStore(mock)
		err := s.HandleStocktakeDiff(context.Background(), 5)
		if !errors.Is(err, ErrDiffPending) {
			t.Fatalf("err=%v, want ErrDiffPending", err)
		}
	})

	t.Run("任务不存在", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		diffCloseMocks(mock, 99, 0)
		mock.ExpectExec(`UPDATE stocktakes`).
			WithArgs(int64(99)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		s := NewPGStore(mock)
		if err := s.HandleStocktakeDiff(context.Background(), 99); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
}

// CreateStocktake 建单快照(TDD 先行):主体校验 + 冻结范围内资产为 PENDING 明细。
func TestPGStore_CreateStocktake(t *testing.T) {
	t.Run("快照范围内资产", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM legal_entities`).
			WithArgs(int64(1)).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`INSERT INTO stocktakes`).
			WithArgs(int64(1), "全库", "DOING").
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(9)))
		mock.ExpectExec(`INSERT INTO stocktake_items`).
			WithArgs(int64(9), int64(1), "全库").
			WillReturnResult(pgxmock.NewResult("INSERT", 40))
		s := NewPGStore(mock)
		id, err := s.CreateStocktake(context.Background(), Stocktake{LegalEntityID: 1, Scope: "全库"})
		if err != nil || id != 9 {
			t.Fatalf("id=%d err=%v", id, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("主体不存在拒绝", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM legal_entities`).
			WithArgs(int64(42)).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
		s := NewPGStore(mock)
		if _, err := s.CreateStocktake(context.Background(), Stocktake{LegalEntityID: 42, Scope: "全库"}); !errors.Is(err, ErrForeignKeyViolation) {
			t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
		}
	})
}

// ScanStocktake 扫码回填(TDD 先行):预期行判 OK/MISMATCH,计划外行 EXTRA,重算进度。
func TestPGStore_ScanStocktake(t *testing.T) {
	scanMocks := func(mock pgxmock.PgxPoolIface, taskID, assetID int64) {
		mock.ExpectQuery(`SELECT status FROM stocktakes WHERE id = \$1`).
			WithArgs(taskID).
			WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("DOING"))
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM assets`).
			WithArgs(assetID).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	}
	t.Run("预期行 MISMATCH 并重算进度", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		scanMocks(mock, 1, 7)
		mock.ExpectExec(`UPDATE stocktake_items SET scanned_status`).
			WithArgs(int64(1), int64(7), "DEPLOYED").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`UPDATE stocktakes SET progress`).
			WithArgs(int64(1)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectQuery(`SELECT id, kind FROM stocktake_items`).
			WithArgs(int64(1), int64(7)).
			WillReturnRows(pgxmock.NewRows([]string{"id", "kind"}).AddRow(int64(3), "MISMATCH"))
		s := NewPGStore(mock)
		id, kind, err := s.ScanStocktake(context.Background(), 1, 7, "DEPLOYED")
		if err != nil || id != 3 || kind != "MISMATCH" {
			t.Fatalf("id=%d kind=%s err=%v", id, kind, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("计划外资产记 EXTRA", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		scanMocks(mock, 1, 8)
		mock.ExpectExec(`UPDATE stocktake_items SET scanned_status`).
			WithArgs(int64(1), int64(8), "IN_STOCK").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		mock.ExpectExec(`INSERT INTO stocktake_items`).
			WithArgs(int64(1), int64(8), "IN_STOCK").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE stocktakes SET progress`).
			WithArgs(int64(1)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectQuery(`SELECT id, kind FROM stocktake_items`).
			WithArgs(int64(1), int64(8)).
			WillReturnRows(pgxmock.NewRows([]string{"id", "kind"}).AddRow(int64(4), "EXTRA"))
		s := NewPGStore(mock)
		_, kind, err := s.ScanStocktake(context.Background(), 1, 8, "IN_STOCK")
		if err != nil || kind != "EXTRA" {
			t.Fatalf("kind=%s err=%v", kind, err)
		}
	})

	t.Run("已关单拒绝扫码", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT status FROM stocktakes WHERE id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("DONE"))
		s := NewPGStore(mock)
		if _, _, err := s.ScanStocktake(context.Background(), 1, 7, "IN_STOCK"); !errors.Is(err, ErrStocktakeState) {
			t.Fatalf("err=%v, want ErrStocktakeState", err)
		}
	})
}

// HandleStocktakeItem 逐条处置(TDD 先行):CONFIRM 修正台账+轨迹,FIX/ESCALATE 留痕。
func TestPGStore_HandleStocktakeItem(t *testing.T) {
	loadOpenDiff := func(mock pgxmock.PgxPoolIface) {
		mock.ExpectQuery(`SELECT status FROM stocktakes WHERE id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("DOING"))
		mock.ExpectQuery(`SELECT id, task_id, asset_id`).
			WithArgs(int64(3), int64(1)).
			WillReturnRows(pgxmock.NewRows([]string{"id", "task_id", "asset_id", "expected_status", "scanned_status", "kind", "resolution"}).
				AddRow(int64(3), int64(1), int64(7), "IN_STOCK", "DEPLOYED", "MISMATCH", "OPEN"))
	}
	t.Run("CONFIRM MISMATCH 修正台账并留轨迹", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		loadOpenDiff(mock)
		mock.ExpectExec(`UPDATE assets SET status`).
			WithArgs(int64(7), "DEPLOYED").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`INSERT INTO asset_lifecycles`).
			WithArgs(int64(7), "DEPLOYED").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE stocktake_items SET resolution`).
			WithArgs(int64(3), int64(1), "CONFIRMED", int64(55), pgxmock.AnyArg(), "").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.HandleStocktakeItem(context.Background(), 1, 3, "CONFIRM", "", 55); err != nil {
			t.Fatalf("HandleStocktakeItem: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("FIX 留痕不改台账", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		loadOpenDiff(mock)
		mock.ExpectExec(`UPDATE stocktake_items SET resolution`).
			WithArgs(int64(3), int64(1), "FIXED", int64(55), pgxmock.AnyArg(), "现场核实台账正确").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.HandleStocktakeItem(context.Background(), 1, 3, "FIX", "现场核实台账正确", 55); err != nil {
			t.Fatalf("HandleStocktakeItem: %v", err)
		}
	})

	t.Run("非法动作/OK行/重复处置拒绝", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		s := NewPGStore(mock)
		if err := s.HandleStocktakeItem(context.Background(), 1, 3, "NOPE", "", 55); !errors.Is(err, ErrStocktakeState) {
			t.Fatalf("action err=%v", err)
		}
		mock.ExpectQuery(`SELECT status FROM stocktakes WHERE id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("DOING"))
		mock.ExpectQuery(`SELECT id, task_id, asset_id`).
			WithArgs(int64(3), int64(1)).
			WillReturnRows(pgxmock.NewRows([]string{"id", "task_id", "asset_id", "expected_status", "scanned_status", "kind", "resolution"}).
				AddRow(int64(3), int64(1), int64(7), "IN_STOCK", "IN_STOCK", "OK", "OPEN"))
		if err := s.HandleStocktakeItem(context.Background(), 1, 3, "CONFIRM", "", 55); !errors.Is(err, ErrStocktakeState) {
			t.Fatalf("ok-line err=%v", err)
		}
	})
}

// ListStocktakeItems 明细清单(TDD 先行)。
func TestPGStore_ListStocktakeItems(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	scanned := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT id, task_id, asset_id`).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "task_id", "asset_id", "expected_status",
			"scanned_status", "scanned_at", "kind", "resolution", "handled_by", "handled_at", "note"}).
			AddRow(int64(1), int64(1), int64(7), "IN_STOCK", "DEPLOYED", scanned, "MISMATCH", "CONFIRMED", int64(55), scanned, ""))
	s := NewPGStore(mock)
	items, err := s.ListStocktakeItems(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListStocktakeItems: %v", err)
	}
	if len(items) != 1 || items[0].Kind != "MISMATCH" || items[0].ScannedAt == nil || items[0].HandledBy != 55 {
		t.Fatalf("items=%+v", items)
	}
}
