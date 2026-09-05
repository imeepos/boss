package asset

// P1-T1 MarkDeployed/MarkReleased 事务面单测:幂等重放/SCRAPPED 拒绝/轨迹落行。

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_MarkDeployed(t *testing.T) {
	t.Run("IN_STOCK → DEPLOYED+绑地址+轨迹行", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT status FROM assets WHERE id = \$1 FOR UPDATE`).
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"status"}).AddRow("IN_STOCK"))
		mock.ExpectExec(`UPDATE assets SET status = 'DEPLOYED'`).
			WithArgs(int64(5), int64(21)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectQuery(`SELECT COALESCE\(name, ''\) FROM addresses WHERE id = \$1`).
			WithArgs(int64(21)).
			WillReturnRows(mock.NewRows([]string{"name"}).AddRow("望京X"))
		mock.ExpectExec(`INSERT INTO asset_lifecycles`).
			WithArgs(int64(5), int64(21), "望京X", int64(2), "张师傅").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := NewPGStore(mock)
		if err := s.MarkDeployed(context.Background(), mock, 5, 21, 2, "张师傅"); err != nil {
			t.Fatalf("MarkDeployed: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("已 DEPLOYED → 幂等 no-op", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT status FROM assets WHERE id = \$1 FOR UPDATE`).
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"status"}).AddRow("DEPLOYED"))

		s := NewPGStore(mock)
		if err := s.MarkDeployed(context.Background(), mock, 5, 21, 2, "张师傅"); err != nil {
			t.Fatalf("MarkDeployed replay: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("SCRAPPED → 拒绝联动转人工", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT status FROM assets WHERE id = \$1 FOR UPDATE`).
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"status"}).AddRow("SCRAPPED"))

		s := NewPGStore(mock)
		err := s.MarkDeployed(context.Background(), mock, 5, 21, 2, "张师傅")
		if !errors.Is(err, ErrAssetScrapped) {
			t.Fatalf("err=%v, want ErrAssetScrapped", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func TestPGStore_MarkReleased(t *testing.T) {
	t.Run("DEPLOYED → IN_STOCK+清地址+轨迹行", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT status FROM assets WHERE id = \$1 FOR UPDATE`).
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"status"}).AddRow("DEPLOYED"))
		mock.ExpectExec(`UPDATE assets SET status = 'IN_STOCK'`).
			WithArgs(int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`INSERT INTO asset_lifecycles`).
			WithArgs(int64(5)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := NewPGStore(mock)
		if err := s.MarkReleased(context.Background(), mock, 5); err != nil {
			t.Fatalf("MarkReleased: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("已 IN_STOCK → 幂等 no-op", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT status FROM assets WHERE id = \$1 FOR UPDATE`).
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"status"}).AddRow("IN_STOCK"))

		s := NewPGStore(mock)
		if err := s.MarkReleased(context.Background(), mock, 5); err != nil {
			t.Fatalf("MarkReleased replay: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}
