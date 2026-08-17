package quadlink

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

var cols = []string{"id", "asset_id", "customer_id", "port_id", "address_id", "legal_entity_id", "legal_entity_name", "status"}

func TestPGStore_ListLinks(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, asset_id, customer_id, port_id, address_id, legal_entity_id, legal_entity_name, status FROM quad_links`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(100), int64(1), int64(200), int64(300), int64(1), "主品牌·企业", "LINKED"))

	s := NewPGStore(mock)
	got, err := s.ListLinks(context.Background())
	if err != nil {
		t.Fatalf("ListLinks: %v", err)
	}
	if len(got) != 1 || got[0].Status != "LINKED" || got[0].CustomerID != 1 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateLink(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO quad_links`).
		WithArgs(int64(101), int64(2), int64(201), int64(301), int64(1), "主品牌·企业", "LINKED").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.CreateLink(context.Background(), QuadLink{
		AssetID: 101, CustomerID: 2, PortID: 201, AddressID: 301, LegalEntityID: 1, LegalEntityName: "主品牌·企业", Status: "LINKED",
	})
	if err != nil {
		t.Fatalf("CreateLink: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_GetByAsset(t *testing.T) {
	t.Run("命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT id, asset_id, customer_id, port_id, address_id, legal_entity_id, legal_entity_name, status FROM quad_links WHERE asset_id = \$1`).
			WithArgs(int64(100)).
			WillReturnRows(mock.NewRows(cols).
				AddRow(int64(1), int64(100), int64(1), int64(200), int64(300), int64(1), "主品牌·企业", "LINKED"))

		s := NewPGStore(mock)
		q, err := s.GetByAsset(context.Background(), 100)
		if err != nil {
			t.Fatalf("GetByAsset: %v", err)
		}
		if q.CustomerID != 1 || q.PortID != 200 {
			t.Fatalf("q=%+v", q)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("未命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`WHERE asset_id = \$1`).
			WithArgs(int64(999)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		_, err = s.GetByAsset(context.Background(), 999)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}
