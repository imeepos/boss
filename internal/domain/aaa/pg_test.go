package aaa

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

var ts = time.Date(2025, 8, 17, 10, 0, 0, 0, time.UTC)

func TestPGStore_ListLoAccounts(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "loid", "customer_id", "legal_entity_id", "legal_entity_name", "region_id", "region_name", "region_path", "offer_id", "qos_template_id", "status", "billing_mode"}
	mock.ExpectQuery(`SELECT id, loid, customer_id, legal_entity_id, legal_entity_name, region_id, region_name, COALESCE\(region_path,''\), offer_id, qos_template_id, status, billing_mode FROM lo_accounts`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "LOID-88A1", int64(1), int64(1), "主品牌·企业", int64(11), "马尼拉市", "root.luzon.ncr.manila", int64(3), int64(1), "ACTIVE", "POSTPAID"))

	s := NewPGStore(mock)
	got, err := s.ListLoAccounts(context.Background())
	if err != nil {
		t.Fatalf("ListLoAccounts: %v", err)
	}
	if len(got) != 1 || got[0].Loid != "LOID-88A1" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateLoAccount(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO lo_accounts`).
		WithArgs("LOID-88A2", int64(4), int64(1), "主品牌·企业", int64(11), "马尼拉市", nil, int64(2), int64(1), "ACTIVE", "PREPAID").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.CreateLoAccount(context.Background(), LoAccount{
		Loid: "LOID-88A2", CustomerID: 4, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		RegionID: 11, RegionName: "马尼拉市", OfferID: 2, QosTemplateID: 1, Status: "ACTIVE",
		BillingMode: BillingModePrepaid,
	})
	if err != nil {
		t.Fatalf("CreateLoAccount: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_GetLoAccountByLoid(t *testing.T) {
	t.Run("命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		cols := []string{"id", "loid", "customer_id", "legal_entity_id", "legal_entity_name", "region_id", "region_name", "region_path", "offer_id", "qos_template_id", "status", "billing_mode"}
		mock.ExpectQuery(`SELECT id, loid, customer_id`).
			WithArgs("LOID-88A1").
			WillReturnRows(mock.NewRows(cols).
				AddRow(int64(1), "LOID-88A1", int64(1), int64(1), "主品牌·企业", int64(11), "马尼拉市", "", int64(3), int64(1), "ACTIVE", "POSTPAID"))

		s := NewPGStore(mock)
		a, err := s.GetLoAccountByLoid(context.Background(), "LOID-88A1")
		if err != nil {
			t.Fatalf("GetLoAccountByLoid: %v", err)
		}
		if a.CustomerID != 1 {
			t.Fatalf("a=%+v", a)
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

		mock.ExpectQuery(`SELECT id, loid`).
			WithArgs("LOID-9999").
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		_, err = s.GetLoAccountByLoid(context.Background(), "LOID-9999")
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func TestPGStore_AppendCdr(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO cdrs`).
		WithArgs("LOID-88A1", "boss", int16(1), "S-1", int32(7200), int64(1024), int64(2048), "10.0.0.1", "UNBILLED").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	s := NewPGStore(mock)
	id, err := s.AppendCdr(context.Background(), CdrRecord{
		Loid: "LOID-88A1", Username: "boss", AcctStatus: 1, SessionID: "S-1", SessionTime: 7200,
		InputOctets: 1024, OutputOctets: 2048, NasIP: "10.0.0.1", BillingStatus: "UNBILLED",
	})
	if err != nil {
		t.Fatalf("AppendCdr: %v", err)
	}
	if id != 1 {
		t.Fatalf("id=%d, want 1", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListCdrs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "loid", "username", "acct_status", "session_id", "session_time", "input_octets", "output_octets", "nas_ip", "billing_status", "started_at"}
	mock.ExpectQuery(`SELECT id, loid, COALESCE\(username, ''\), acct_status, COALESCE\(session_id, ''\), session_time, input_octets, output_octets, COALESCE\(nas_ip, ''\), billing_status, started_at`).
		WithArgs("LOID-88A1").
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "LOID-88A1", "boss", int16(1), "S-1", int32(7200), int64(1024), int64(2048), "10.0.0.1", "UNBILLED", ts))

	s := NewPGStore(mock)
	got, err := s.ListCdrs(context.Background(), "LOID-88A1")
	if err != nil {
		t.Fatalf("ListCdrs: %v", err)
	}
	if len(got) != 1 || got[0].SessionTime != 7200 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AppendAuthLog(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO auth_logs`).
		WithArgs("LOID-88A1", "SUCCESS").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	s := NewPGStore(mock)
	id, err := s.AppendAuthLog(context.Background(), AuthLog{Loid: "LOID-88A1", Result: "SUCCESS"})
	if err != nil {
		t.Fatalf("AppendAuthLog: %v", err)
	}
	if id != 1 {
		t.Fatalf("id=%d, want 1", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListAuthLogs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, loid, result, created_at FROM auth_logs`).
		WithArgs("LOID-88A1").
		WillReturnRows(mock.NewRows([]string{"id", "loid", "result", "created_at"}).
			AddRow(int64(1), "LOID-88A1", "SUCCESS", ts))

	s := NewPGStore(mock)
	got, err := s.ListAuthLogs(context.Background(), "LOID-88A1")
	if err != nil {
		t.Fatalf("ListAuthLogs: %v", err)
	}
	if len(got) != 1 || got[0].Result != "SUCCESS" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPGStore_GetLoAccountByCustomer 契约:按客户查 1:1 LO 账号;未命中 ErrNotFound。
func TestPGStore_GetLoAccountByCustomer(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "loid", "customer_id", "legal_entity_id", "legal_entity_name", "region_id", "region_name", "region_path", "offer_id", "qos_template_id", "status", "billing_mode"}
	mock.ExpectQuery(`SELECT id, loid, customer_id`).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "LOID-88A1", int64(9), int64(1), "主品牌·企业", int64(11), "马尼拉市", "", int64(3), int64(1), "ACTIVE", "POSTPAID"))

	s := NewPGStore(mock)
	a, err := s.GetLoAccountByCustomer(context.Background(), 9)
	if err != nil {
		t.Fatalf("GetLoAccountByCustomer: %v", err)
	}
	if a.Loid != "LOID-88A1" || a.CustomerID != 9 {
		t.Fatalf("a=%+v", a)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPGStore_CreateLoAccountBillingModeDefault 契约:BillingMode 空回退 POSTPAID(000102)。
func TestPGStore_CreateLoAccountBillingModeDefault(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO lo_accounts`).
		WithArgs("LOID-88B1", int64(5), int64(1), "主品牌·企业", int64(11), "马尼拉市", nil, int64(2), int64(1), "ACTIVE", "POSTPAID").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))

	s := NewPGStore(mock)
	if _, err := s.CreateLoAccount(context.Background(), LoAccount{
		Loid: "LOID-88B1", CustomerID: 5, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		RegionID: 11, RegionName: "马尼拉市", OfferID: 2, QosTemplateID: 1, Status: "ACTIVE",
	}); err != nil {
		t.Fatalf("CreateLoAccount: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
