package customer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

var fixedTime = time.Date(2025, 8, 17, 10, 0, 0, 0, time.UTC)

// TestPGStore_Create 契约:建档并返回自增 id。
func TestPGStore_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// FK validation: address exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(100)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	// FK validation: legal entity exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	// NULLIF($7,0) 锁死:0 地址落 NULL 绕 FK(000176 起可空),回归防护。
	mock.ExpectQuery(`INSERT INTO customers.*NULLIF\(\$7,0\)`).
		WithArgs("王先生", "13800001111", "身份证", "110101199001011234", "VERIFIED", "ACTIVE",
			int64(100), int64(1), int64(11), "root.luzon.ncr.manila").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(7)))

	s := NewPGStore(mock)
	id, err := s.Create(context.Background(), Customer{
		Name: "王先生", Phone: "13800001111", IdType: "身份证", IdNo: "110101199001011234",
		RealNameStatus: "VERIFIED", ServiceStatus: "ACTIVE",
		AddressID: 100, LegalEntityID: 1, RegionID: 11, RegionName: "root.luzon.ncr.manila",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id != 7 {
		t.Fatalf("id=%d, want 7", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_Get 契约:按 id 查客户;未命中返回 ErrCustomerNotFound。
func TestPGStore_Get(t *testing.T) {
	t.Run("命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT customers.id, customers.customer_code, customers.name, customers.phone, customers.id_type, customers.id_no, customers.real_name_status, customers.service_status, COALESCE\(customers.address_id, 0\) AS address_id, customers.legal_entity_id, COALESCE\(le.name, ''\) AS legal_entity_name, customers.region_id, customers.region_name, customers.created_at FROM customers LEFT JOIN legal_entities`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{
				"id", "customer_code", "name", "phone", "id_type", "id_no", "real_name_status", "service_status",
				"address_id", "legal_entity_id", "legal_entity_name", "region_id", "region_name", "created_at",
			}).AddRow(int64(1), "C-00000001", "王先生", "13800001111", "身份证", "110101199001011234", "VERIFIED", "ACTIVE",
				int64(100), int64(1), "Smoke Partner Co", int64(11), "root.luzon.ncr.manila", fixedTime))

		s := NewPGStore(mock)
		c, err := s.Get(context.Background(), 1)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if c.CustomerCode != "C-00000001" {
			t.Fatalf("CustomerCode=%q, want C-00000001", c.CustomerCode)
		}
		if c.LegalEntityName != "Smoke Partner Co" {
			t.Fatalf("LegalEntityName=%q, want Smoke Partner Co", c.LegalEntityName)
		}
		if c.Name != "王先生" || c.Phone != "13800001111" || c.RegionID != 11 {
			t.Fatalf("c=%+v", c)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
	t.Run("未命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT customers.id, customers.customer_code`).
			WithArgs(int64(99)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		_, err = s.Get(context.Background(), 99)
		if !errors.Is(err, ErrCustomerNotFound) {
			t.Fatalf("err=%v, want ErrCustomerNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

// TestPGStore_List 契约:按关键字/电话/状态过滤 + 分页。
func TestPGStore_List(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT customers.id, customers.customer_code, customers.name, customers.phone`).
		WithArgs("王", "", "", int64(0), "", 10, 0).
		WillReturnRows(mock.NewRows([]string{
			"id", "customer_code", "name", "phone", "id_type", "id_no", "real_name_status", "service_status",
			"address_id", "legal_entity_id", "legal_entity_name", "region_id", "region_name", "created_at",
		}).AddRow(int64(1), "C-00000001", "王先生", "13800001111", "身份证", "110101199001011234", "VERIFIED", "ACTIVE",
			int64(100), int64(1), "Smoke Partner Co", int64(11), "root.luzon.ncr.manila", fixedTime))

	s := NewPGStore(mock)
	got, err := s.List(context.Background(), CustomerQuery{NameKeyword: "王", Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].Name != "王先生" || got[0].CustomerCode != "C-00000001" {
		t.Fatalf("got=%+v", got)
	}
	if got[0].LegalEntityName != "Smoke Partner Co" {
		t.Fatalf("LegalEntityName=%q, want Smoke Partner Co", got[0].LegalEntityName)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
