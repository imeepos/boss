package user

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_ListLegalEntities 契约:返回全部子公司,按 id 升序。
func TestPGStore_ListLegalEntities(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, code, name FROM legal_entities ORDER BY id`).
		WillReturnRows(mock.NewRows([]string{"id", "code", "name"}).
			AddRow(int64(1), "LEG-A", "主品牌·企业").
			AddRow(int64(2), "LEG-B", "家庭宽带").
			AddRow(int64(3), "LEG-C", "批发品牌"))

	s := NewPGStore(mock)
	got, err := s.ListLegalEntities(context.Background())
	if err != nil {
		t.Fatalf("ListLegalEntities: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len=%d, want 3", len(got))
	}
	if got[0].Code != "LEG-A" || got[1].Code != "LEG-B" || got[2].Code != "LEG-C" {
		t.Fatalf("codes=%+v, want LEG-A/B/C", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_ListRegions 契约:返回全部经营区域,并派生父路径。
func TestPGStore_ListRegions(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, path, level, name FROM regions ORDER BY path`).
		WillReturnRows(mock.NewRows([]string{"id", "path", "level", "name"}).
			AddRow(int64(1), "root", int8(1), "集团").
			AddRow(int64(2), "root.luzon", int8(2), "吕宋大区"))

	s := NewPGStore(mock)
	got, err := s.ListRegions(context.Background(), "")
	if err != nil {
		t.Fatalf("ListRegions: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if got[0].Parent != "" {
		t.Fatalf("root.Parent=%q, want empty", got[0].Parent)
	}
	if got[1].Parent != "root" {
		t.Fatalf("root.luzon.Parent=%q, want root", got[1].Parent)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
