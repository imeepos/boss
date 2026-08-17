package order

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_ListDispatchTickets(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "ticket_no", "order_id", "worker_id", "worker_name", "group_id", "group_name", "region_id", "region_name", "legal_entity_id", "legal_entity_name", "status"}
	mock.ExpectQuery(`SELECT id, ticket_no, order_id, COALESCE\(worker_id, 0\)`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "WO-01", int64(1), int64(1024), "张师傅", int64(1), "装机一组", int64(11), "马尼拉市", int64(1), "主品牌·企业", "DOING"))

	s := NewPGStore(mock, stubExists{})
	got, err := s.ListDispatchTickets(context.Background())
	if err != nil {
		t.Fatalf("ListDispatchTickets: %v", err)
	}
	if len(got) != 1 || got[0].WorkerID != 1024 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateDispatchTicket(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO dispatch_tickets`).
		WithArgs("WO-02", int64(2), nil, "", nil, "", nil, "", int64(1), "主品牌·企业", "PENDING").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock, stubExists{})
	id, err := s.CreateDispatchTicket(context.Background(), DispatchTicket{
		TicketNo: "WO-02", OrderID: 2, LegalEntityID: 1, LegalEntityName: "主品牌·企业", Status: "PENDING",
	})
	if err != nil {
		t.Fatalf("CreateDispatchTicket: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListComplaints(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "ticket_no", "customer_id", "order_id", "legal_entity_id", "legal_entity_name", "type", "status"}
	mock.ExpectQuery(`SELECT id, ticket_no, customer_id, COALESCE\(order_id, 0\)`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "TKT-20250817-012", int64(8), int64(0), int64(1), "主品牌·企业", "no_internet", "PROCESSING"))

	s := NewPGStore(mock, stubExists{})
	got, err := s.ListComplaints(context.Background())
	if err != nil {
		t.Fatalf("ListComplaints: %v", err)
	}
	if len(got) != 1 || got[0].TicketNo != "TKT-20250817-012" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateComplaint(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO complaints`).
		WithArgs("TKT-20250817-013", int64(1), nil, int64(1), "主品牌·企业", "slow", "OPEN").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock, stubExists{})
	id, err := s.CreateComplaint(context.Background(), Complaint{
		TicketNo: "TKT-20250817-013", CustomerID: 1, LegalEntityID: 1, LegalEntityName: "主品牌·企业", Type: "slow", Status: "OPEN",
	})
	if err != nil {
		t.Fatalf("CreateComplaint: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListScanLogs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "order_id", "worker_id", "worker_name", "tag_id", "result"}
	mock.ExpectQuery(`SELECT id, order_id, worker_id, worker_name, tag_id, result FROM scan_logs`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), int64(1024), "张师傅", int64(1001), "MATCH"))

	s := NewPGStore(mock, stubExists{})
	got, err := s.ListScanLogs(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListScanLogs: %v", err)
	}
	if len(got) != 1 || got[0].Result != "MATCH" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AppendScanLog(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO scan_logs`).
		WithArgs(int64(1), int64(1024), "张师傅", int64(1001), "MISMATCH").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock, stubExists{})
	id, err := s.AppendScanLog(context.Background(), ScanLog{
		OrderID: 1, WorkerID: 1024, WorkerName: "张师傅", TagID: 1001, Result: "MISMATCH",
	})
	if err != nil {
		t.Fatalf("AppendScanLog: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// GetDispatchTicketByNo 扫码闭环入口:ticketNo → 工单(含 orderID)。
func TestPGStore_GetDispatchTicketByNo(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`FROM dispatch_tickets`).
		WithArgs("TIC-1").
		WillReturnRows(mock.NewRows([]string{"id", "ticket_no", "order_id", "worker_id", "worker_name",
			"group_id", "group_name", "region_id", "region_name", "legal_entity_id", "legal_entity_name", "status"}).
			AddRow(int64(1), "TIC-1", int64(7), int64(2), "张师傅", int64(0), "", int64(0), "", int64(1), "主品牌", "DOING"))

	s := NewPGStore(mock, stubExists{ok: true})
	tk, err := s.GetDispatchTicketByNo(context.Background(), "TIC-1")
	if err != nil {
		t.Fatalf("GetDispatchTicketByNo: %v", err)
	}
	if tk.OrderID != 7 || tk.WorkerID != 2 {
		t.Fatalf("tk=%+v", tk)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
