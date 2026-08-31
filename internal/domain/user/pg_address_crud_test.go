package user

import (
	"context"
	"errors"
	"fmt"
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

// TestSetAddressGeom 契约:范围越界先拒;合法坐标落 ST_MakePoint(lng,lat);未命中 ErrNotFound。
func TestSetAddressGeom(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	s := NewPGStore(mock)
	if err := s.SetAddressGeom(context.Background(), 1, 91, 120); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("lat=91: got %v, want ErrInvalidInput", err)
	}
	if err := s.SetAddressGeom(context.Background(), 1, 14.599, -181); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("lng=-181: got %v, want ErrInvalidInput", err)
	}

	mock.ExpectExec(`UPDATE addresses SET geom = ST_SetSRID\(ST_MakePoint\(\$2, \$3\), 4326\)::geography`).
		WithArgs(int64(3), 120.984, 14.599).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	if err := s.SetAddressGeom(context.Background(), 3, 14.599, 120.984); err != nil {
		t.Fatalf("set geom: %v", err)
	}

	mock.ExpectExec(`UPDATE addresses SET geom`).
		WithArgs(int64(404), 120.984, 14.599).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	if err := s.SetAddressGeom(context.Background(), 404, 14.599, 120.984); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: got %v, want ErrNotFound", err)
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

	mock.ExpectQuery(`LIMIT 21`).
		WithArgs("%朝阳%").
		WillReturnRows(mock.NewRows([]string{"id", "parent_id", "level", "name", "path", "country_code", "admin_code", "has_children"}).
			AddRow(int64(3), int64(2), int8(2), "朝阳区", "bj.chaoyang", "CN", "CN-BJ", false))
	mock.ExpectQuery(`= ANY`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"id", "parent_id", "level", "name", "path", "has_children"}).
			AddRow(int64(1), int64(0), int8(1), "北京市", "bj", true))

	s := NewPGStore(mock)
	hits, hasMore, err := s.SearchAddresses(context.Background(), "朝阳")
	if err != nil {
		t.Fatalf("SearchAddresses: %v", err)
	}
	if hasMore {
		t.Fatalf("single hit must not set hasMore")
	}
	if len(hits) != 1 || hits[0].Node.Name != "朝阳区" {
		t.Fatalf("hits=%+v", hits)
	}
	if len(hits[0].Ancestors) != 1 || hits[0].Ancestors[0].Name != "北京市" {
		t.Fatalf("ancestors=%+v", hits[0].Ancestors)
	}
	if !hits[0].Ancestors[0].HasChildren {
		t.Fatalf("ancestor hasChildren not backfilled: %+v", hits[0].Ancestors[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestSearchAddresses_HasMore 契约:21 条命中取前 20 条并置 hasMore(截断可感知)。
func TestSearchAddresses_HasMore(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := mock.NewRows([]string{"id", "parent_id", "level", "name", "path", "country_code", "admin_code", "has_children"})
	for i := 0; i < 21; i++ {
		rows.AddRow(int64(i+1), int64(0), int8(1), fmt.Sprintf("n%d", i), fmt.Sprintf("p%d", i), "CN", "CN-BJ", false)
	}
	mock.ExpectQuery(`LIMIT 21`).WithArgs("%x%").WillReturnRows(rows)
	mock.ExpectQuery(`= ANY`).WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"id", "parent_id", "level", "name", "path", "has_children"}))

	s := NewPGStore(mock)
	hits, hasMore, err := s.SearchAddresses(context.Background(), "x")
	if err != nil {
		t.Fatalf("SearchAddresses: %v", err)
	}
	if !hasMore || len(hits) != 20 {
		t.Fatalf("hasMore=%v len=%d, want true/20", hasMore, len(hits))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestLookupAddresses 契约:命中保入参顺序;缺失路径进 missing 不报错;祖先链同 search。
func TestLookupAddresses(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`= ANY`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"id", "parent_id", "level", "name", "path", "country_code", "admin_code", "has_children"}).
			AddRow(int64(3), int64(2), int8(2), "朝阳区", "bj.chaoyang", "CN", "CN-BJ", true))
	mock.ExpectQuery(`= ANY`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"id", "parent_id", "level", "name", "path", "has_children"}).
			AddRow(int64(1), int64(0), int8(1), "北京市", "bj", true))

	s := NewPGStore(mock)
	hits, missing, err := s.LookupAddresses(context.Background(),
		[]string{"bj.chaoyang", "bj.gone", "bj.chaoyang"})
	if err != nil {
		t.Fatalf("LookupAddresses: %v", err)
	}
	if len(hits) != 1 || hits[0].Node.Path != "bj.chaoyang" || !hits[0].Node.HasChildren {
		t.Fatalf("hits=%+v", hits)
	}
	if len(hits[0].Ancestors) != 1 || hits[0].Ancestors[0].Name != "北京市" {
		t.Fatalf("ancestors=%+v", hits[0].Ancestors)
	}
	if len(missing) != 1 || missing[0] != "bj.gone" {
		t.Fatalf("missing=%+v", missing)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
