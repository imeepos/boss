package asset

// P1-T2 事件流与回收闭环单测:解绑/报废的状态+事件原子性,幂等与冲突路径。
// pgxmock 正则为子串匹配,锚点一律用无特殊字符的短片段(避 $/() 转义)。

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_UnbindTag(t *testing.T) {
	t.Run("已绑定 → 解绑+UNBIND 事件", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(`bound_asset_id, 0. FROM tags WHERE id`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"bound_asset_id"}).AddRow(int64(5)))
		mock.ExpectExec(`UPDATE tags SET bound_asset_id = NULL`).
			WithArgs(int64(1), int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`INSERT INTO tag_events`).
			WithArgs(int64(1), int64(5), nil, "换新回收", pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		s := NewPGStore(mock)
		if err := s.UnbindTag(context.Background(), 1, 0, 0, "换新回收"); err != nil {
			t.Fatalf("UnbindTag: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("未绑定 → ErrTagUnbound", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(`bound_asset_id, 0. FROM tags WHERE id`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"bound_asset_id"}).AddRow(int64(0)))
		mock.ExpectRollback()

		s := NewPGStore(mock)
		if err := s.UnbindTag(context.Background(), 1, 0, 0, ""); !errors.Is(err, ErrTagUnbound) {
			t.Fatalf("err=%v, want ErrTagUnbound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("预期资产不符 → ErrBindingConflict", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(`bound_asset_id, 0. FROM tags WHERE id`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"bound_asset_id"}).AddRow(int64(5)))
		mock.ExpectRollback()

		s := NewPGStore(mock)
		if err := s.UnbindTag(context.Background(), 1, 9, 0, ""); !errors.Is(err, ErrBindingConflict) {
			t.Fatalf("err=%v, want ErrBindingConflict", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func TestPGStore_ScrapAsset(t *testing.T) {
	t.Run("IN_STOCK 带标签 → SCRAPPED+轨迹+标签 RECYCLE", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT status, COALESCE.tag_id, 0. FROM assets WHERE id`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"status", "tag_id"}).AddRow("IN_STOCK", int64(9)))
		mock.ExpectExec(`UPDATE assets SET status = 'SCRAPPED'`).
			WithArgs(int64(7)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`INSERT INTO asset_lifecycles`).
			WithArgs(int64(7)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE tags SET bound_asset_id = NULL`).
			WithArgs(int64(9), int64(7)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`INSERT INTO tag_events`).
			WithArgs(int64(9), int64(7), int64(3), "屏裂报废", pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		s := NewPGStore(mock)
		if err := s.ScrapAsset(context.Background(), 7, 3, "屏裂报废"); err != nil {
			t.Fatalf("ScrapAsset: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("已 SCRAPPED → 幂等 no-op", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT status, COALESCE.tag_id, 0. FROM assets WHERE id`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"status", "tag_id"}).AddRow("SCRAPPED", int64(0)))
		mock.ExpectRollback()

		s := NewPGStore(mock)
		if err := s.ScrapAsset(context.Background(), 7, 3, ""); err != nil {
			t.Fatalf("ScrapAsset replay: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}
