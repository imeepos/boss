package worker

// 师傅负责区域(000175)域层测试:覆盖式配置/主区域回写/区域校验/匹配口径。

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestNormalizeRegionIDs(t *testing.T) {
	ids, primary := NormalizeRegionIDs([]int64{12, 0, 11, 12, -3}, 0)
	if primary != 12 || len(ids) != 2 || ids[0] != 12 || ids[1] != 11 {
		t.Fatalf("got ids=%v primary=%d, want [12 11] 12", ids, primary)
	}
	// 主区域在集合中:前提首位。
	ids, primary = NormalizeRegionIDs([]int64{12, 11}, 11)
	if primary != 11 || ids[0] != 11 || ids[1] != 12 {
		t.Fatalf("got ids=%v primary=%d, want [11 12] 11", ids, primary)
	}
	// 主区域不在集合中:取首个为主。
	ids, primary = NormalizeRegionIDs([]int64{12, 11}, 99)
	if primary != 12 || ids[0] != 12 {
		t.Fatalf("got ids=%v primary=%d, want primary=12", ids, primary)
	}
	// 全非法:空集,主区域保留。
	ids, primary = NormalizeRegionIDs([]int64{0, -1}, 7)
	if ids != nil || primary != 7 {
		t.Fatalf("got ids=%v primary=%d, want nil/7", ids, primary)
	}
}

func TestWorkerMatchesRegion(t *testing.T) {
	cases := []struct {
		name   string
		w      Worker
		ticket int64
		want   bool
	}{
		{"工单无区域=不限", Worker{RegionID: 1}, 0, true},
		{"师傅无区域=不限", Worker{}, 5, true},
		{"主区域命中", Worker{RegionID: 1}, 1, true},
		{"主区域不命中", Worker{RegionID: 1}, 2, false},
		{"扩展区域命中", Worker{RegionID: 1, RegionIDs: []int64{1, 2}}, 2, true},
		{"集合外不命中", Worker{RegionID: 1, RegionIDs: []int64{1, 2}}, 3, false},
	}
	for _, tc := range cases {
		if got := tc.w.MatchesRegion(tc.ticket); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestPGStore_SetWorkerRegions(t *testing.T) {
	t.Run("覆盖式配置+主区域回写", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()

		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		// 输入 [12 11] 无显式主区域 → 首位 12 为主,按序校验/落表。
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(12)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(11)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM worker_regions`).
			WithArgs(int64(5)).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectExec(`INSERT INTO worker_regions`).
			WithArgs(int64(5), int64(12)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`INSERT INTO worker_regions`).
			WithArgs(int64(5), int64(11)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE workers SET region_id`).
			WithArgs(int64(5), int64(12)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		s := NewPGStore(mock)
		if err := s.SetWorkerRegions(context.Background(), 5, []int64{12, 11}); err != nil {
			t.Fatalf("SetWorkerRegions: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("区域不存在", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(99)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

		s := NewPGStore(mock)
		err := s.SetWorkerRegions(context.Background(), 5, []int64{99})
		if !errors.Is(err, ErrForeignKeyViolation) {
			t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
		}
	})
	t.Run("师傅不存在", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(9)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

		s := NewPGStore(mock)
		if err := s.SetWorkerRegions(context.Background(), 9, []int64{1}); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
}
