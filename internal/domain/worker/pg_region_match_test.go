package worker

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestMatchedRegionIDs 契约:师傅区域 0 全放行不查库;工单区域 0 放行;
// 非零工单区域走 ltree 子树判定,未命中/未返回的 id 判 false。
func TestMatchedRegionIDs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	s := NewPGStore(mock)

	t.Run("师傅无区域全放行", func(t *testing.T) {
		got, err := s.MatchedRegionIDs(context.Background(), 0, []int64{3, 8})
		if err != nil || !got[3] || !got[8] {
			t.Fatalf("got=%v err=%v", got, err)
		}
	})

	t.Run("工单无区域放行+其余查库", func(t *testing.T) {
		mock.ExpectQuery(`path <@ \(SELECT path FROM regions WHERE id = \$2\)`).
			WithArgs([]int64{3, 8}, int64(2)).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))
		got, err := s.MatchedRegionIDs(context.Background(), 2, []int64{0, 3, 8})
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if !got[0] || !got[3] || got[8] {
			t.Fatalf("子树判定失真: got=%v", got)
		}
	})
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
