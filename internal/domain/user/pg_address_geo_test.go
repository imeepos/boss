package user

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_SetAddressGeo 契约:仅根节点可挂接;非根返回 ErrInvalidInput,不存在返回 ErrNotFound。
func TestPGStore_SetAddressGeo(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT level FROM addresses`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"level"}).AddRow(int8(2)))

	s := NewPGStore(mock)
	if err := s.SetAddressGeo(context.Background(), 7, "CN", "CN-BJ"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("non-root: got %v, want ErrInvalidInput", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_ListUnlinkedRoots 契约:仅返回未挂国家的根节点。
func TestPGStore_ListUnlinkedRoots(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`WHERE level = 1 AND country_code IS NULL`).
		WillReturnRows(mock.NewRows([]string{"id", "parent_id", "level", "name", "country_code", "admin_code"}).
			AddRow(int64(11), int64(0), int8(1), "深圳市", "", ""))

	s := NewPGStore(mock)
	got, err := s.ListUnlinkedRoots(context.Background())
	if err != nil {
		t.Fatalf("ListUnlinkedRoots: %v", err)
	}
	if len(got) != 1 || got[0].Name != "深圳市" || got[0].CountryCode != "" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
