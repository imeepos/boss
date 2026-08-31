package worker

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestMatchedRegionIDs 契约:师傅无任何区域全放行不查库;工单区域 0 放行;
// 非零工单区域走 ltree 子树判定(候选根=主区域 ∪ 扩展区域,000175),未命中判 false。
func TestMatchedRegionIDs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	s := NewPGStore(mock)

	t.Run("师傅无区域全放行", func(t *testing.T) {
		got, err := s.MatchedRegionIDs(context.Background(), &Worker{}, []int64{3, 8})
		if err != nil || !got[3] || !got[8] {
			t.Fatalf("got=%v err=%v", got, err)
		}
	})

	t.Run("工单无区域放行+其余查库", func(t *testing.T) {
		roots := []int64{2, 5}
		mock.ExpectQuery(`t\.path <@ ANY\(SELECT r\.path FROM regions r WHERE r\.id = ANY\(\$2\)\)`).
			WithArgs([]int64{3, 8}, roots).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))
		got, err := s.MatchedRegionIDs(context.Background(),
			&Worker{RegionID: 2, RegionIDs: []int64{5}}, []int64{0, 3, 8})
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

// TestCandidateRegionIDs 契约:主区域在前,扩展区域去 0/去重;nil 安全返回空。
func TestCandidateRegionIDs(t *testing.T) {
	if got := candidateRegionIDs(nil); got != nil {
		t.Fatalf("nil worker: %v", got)
	}
	if got := candidateRegionIDs(&Worker{}); len(got) != 0 {
		t.Fatalf("no regions: %v", got)
	}
	got := candidateRegionIDs(&Worker{RegionID: 2, RegionIDs: []int64{0, 2, 5, 7}})
	if len(got) != 3 || got[0] != 2 || got[1] != 5 || got[2] != 7 {
		t.Fatalf("roots=%v", got)
	}
}
