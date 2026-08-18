package user

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestCreateAddress 契约:label 非法即拒;合法根节点带锚点落库。
func TestCreateAddress(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	s := NewPGStore(mock)
	if _, err := s.CreateAddress(context.Background(), 0, "Bad Label", "x", "", ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("bad label: got %v, want ErrInvalidInput", err)
	}

	mock.ExpectQuery(`INSERT INTO addresses`).
		WithArgs("sz", int8(1), "深圳市", int64(0), "CN", "").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(9)))
	id, err := s.CreateAddress(context.Background(), 0, "sz", "深圳市", "CN", "")
	if err != nil || id != 9 {
		t.Fatalf("create root: id=%d err=%v", id, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestDeleteAddress 契约:有子节点返回 ErrConflict。
func TestDeleteAddress(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(5)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	s := NewPGStore(mock)
	if err := s.DeleteAddress(context.Background(), 5); !errors.Is(err, ErrConflict) {
		t.Fatalf("got %v, want ErrConflict", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestSearchAddresses 契约:命中带按 level 升序的祖先链。
func TestSearchAddresses(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`LIMIT 20`).
		WithArgs("%朝阳%").
		WillReturnRows(mock.NewRows([]string{"id", "parent_id", "level", "name", "path", "country_code", "admin_code"}).
			AddRow(int64(3), int64(2), int8(2), "朝阳区", "bj.chaoyang", "CN", "CN-BJ"))
	mock.ExpectQuery(`= ANY`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"id", "parent_id", "level", "name", "path"}).
			AddRow(int64(1), int64(0), int8(1), "北京市", "bj"))

	s := NewPGStore(mock)
	hits, err := s.SearchAddresses(context.Background(), "朝阳")
	if err != nil {
		t.Fatalf("SearchAddresses: %v", err)
	}
	if len(hits) != 1 || hits[0].Node.Name != "朝阳区" {
		t.Fatalf("hits=%+v", hits)
	}
	if len(hits[0].Ancestors) != 1 || hits[0].Ancestors[0].Name != "北京市" {
		t.Fatalf("ancestors=%+v", hits[0].Ancestors)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
