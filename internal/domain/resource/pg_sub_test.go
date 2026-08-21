package resource

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

var ts = time.Date(2025, 8, 17, 10, 0, 0, 0, time.UTC)

func TestPGStore_ListTransfers(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "transfer_no", "resource_id", "legal_entity_id", "legal_entity_name", "from_region_id", "to_region_id", "status"}
	mock.ExpectQuery(`SELECT id, transfer_no, resource_id, legal_entity_id, legal_entity_name, from_region_id, to_region_id, status FROM transfers`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "TRF-20250817-001", int64(1), int64(1), "主品牌·企业", int64(11), int64(12), "PENDING"))

	s := NewPGStore(mock)
	got, err := s.ListTransfers(context.Background())
	if err != nil {
		t.Fatalf("ListTransfers: %v", err)
	}
	if len(got) != 1 || got[0].TransferNo != "TRF-20250817-001" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateTransfer(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// FK validation: resource exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(2)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`INSERT INTO transfers`).
		WithArgs("TRF-20250817-002", int64(2), int64(1), "主品牌·企业", int64(11), int64(13), "PENDING").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.CreateTransfer(context.Background(), Transfer{
		TransferNo: "TRF-20250817-002", ResourceID: 2, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		FromRegionID: 11, ToRegionID: 13, Status: "PENDING",
	})
	if err != nil {
		t.Fatalf("CreateTransfer: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListExpansions(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "legal_entity_id", "expansion_no", "region_id", "expected_ports", "status"}
	mock.ExpectQuery(`SELECT id, legal_entity_id, expansion_no, region_id, expected_ports, status FROM expansions`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), "EXP-20250817-001", int64(11), int32(100), "PENDING"))

	s := NewPGStore(mock)
	got, err := s.ListExpansions(context.Background())
	if err != nil {
		t.Fatalf("ListExpansions: %v", err)
	}
	if len(got) != 1 || got[0].ExpectedPorts != 100 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateExpansion(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// FK validation: legal entity exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`INSERT INTO expansions`).
		WithArgs(int64(1), "EXP-20250817-002", int64(12), int32(200), "PENDING").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.CreateExpansion(context.Background(), Expansion{
		LegalEntityID: 1, ExpansionNo: "EXP-20250817-002", RegionID: 12, ExpectedPorts: 200, Status: "PENDING",
	})
	if err != nil {
		t.Fatalf("CreateExpansion: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListReserveRecords(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "port_id", "order_id", "status"}
	mock.ExpectQuery(`SELECT id, port_id, order_id, status FROM reserve_records`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), int64(1001), "HELD"))

	s := NewPGStore(mock)
	got, err := s.ListReserveRecords(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListReserveRecords: %v", err)
	}
	if len(got) != 1 || got[0].Status != "HELD" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AppendReserveRecord(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO reserve_records`).
		WithArgs(int64(1), int64(1002), "RELEASED").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.AppendReserveRecord(context.Background(), ReserveRecord{PortID: 1, OrderID: 1002, Status: "RELEASED"})
	if err != nil {
		t.Fatalf("AppendReserveRecord: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListPortHistory(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "port_id", "status", "order_id", "changed_at"}
	mock.ExpectQuery(`SELECT id, port_id, status, COALESCE\(order_id, 0\), changed_at`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), "RESERVED", int64(1001), ts))

	s := NewPGStore(mock)
	got, err := s.ListPortHistory(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListPortHistory: %v", err)
	}
	if len(got) != 1 || got[0].OrderID != 1001 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AppendPortHistory(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO port_change_history`).
		WithArgs(int64(1), "USED", int64(1001), ts).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.AppendPortHistory(context.Background(), PortChangeHistory{
		PortID: 1, Status: "USED", OrderID: 1001, ChangedAt: ts,
	})
	if err != nil {
		t.Fatalf("AppendPortHistory: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
