package order

import (
	"context"
	"errors"
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

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE id = \$1\)`).
		WithArgs(int64(2)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
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

	cols := []string{"id", "ticket_no", "customer_id", "order_id", "legal_entity_id", "legal_entity_name", "type", "status", "created_at", "remote_diagnosis", "sla_deadline", "closed_at", "closed_by", "resolution"}
	mock.ExpectQuery(`SELECT id, ticket_no, customer_id, COALESCE\(order_id, 0\)`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "TKT-20250817-012", int64(8), int64(0), int64(1), "主品牌·企业", "SINGLE_OUTAGE", "PROCESSING", "2025-08-17 10:00", "", "", nil, int64(0), ""))

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

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM customers WHERE id = \$1\)`).
		WithArgs(int64(1)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`INSERT INTO complaints`).
		WithArgs("TKT-20250817-013", int64(1), nil, int64(1), "主品牌·企业", "SLOW_NET", "OPEN", "", "").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock, stubExists{})
	id, err := s.CreateComplaint(context.Background(), Complaint{
		TicketNo: "TKT-20250817-013", CustomerID: 1, LegalEntityID: 1, LegalEntityName: "主品牌·企业", Type: "SLOW_NET", Status: "OPEN",
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

// TestPGStore_CreateComplaint_DerivesCustomerFromOrder 契约:工单域投诉未带
// customer_id 时经订单推导归属(师傅端投诉只带 tk.OrderID,流程数据完善)。
func TestPGStore_CreateComplaint_DerivesCustomerFromOrder(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT customer_id FROM orders WHERE id = \$1`).
		WithArgs(int64(77)).WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(213)))
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM customers WHERE id = \$1\)`).
		WithArgs(int64(213)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE id = \$1\)`).
		WithArgs(int64(77)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`INSERT INTO complaints`).
		WithArgs("TKT-DERIVED", int64(213), int64(77), int64(1), "主品牌·企业", "SLOW_NET", "OPEN", "", "").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(9)))

	s := NewPGStore(mock, stubExists{})
	id, err := s.CreateComplaint(context.Background(), Complaint{
		TicketNo: "TKT-DERIVED", OrderID: 77, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		Type: "SLOW_NET", Status: "OPEN",
	})
	if err != nil {
		t.Fatalf("CreateComplaint: %v", err)
	}
	if id != 9 {
		t.Fatalf("id=%d, want 9", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPGStore_CreateComplaint_RejectsNoAnchor 契约:无客户且无订单时拒建投诉。
func TestPGStore_CreateComplaint_RejectsNoAnchor(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	_, err := NewPGStore(mock, stubExists{}).CreateComplaint(context.Background(), Complaint{
		TicketNo: "TKT-X", LegalEntityID: 1, LegalEntityName: "主品牌·企业", Type: "SLOW_NET", Status: "OPEN",
	})
	if !errors.Is(err, ErrForeignKeyViolation) {
		t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
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

	// 关联完整性:order 存在性校验。
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders`).
		WithArgs(int64(1)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
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

// TestPGStore_AppendScanLog_RejectsOrphanOrder 契约:order 缺失拒写扫码日志。
func TestPGStore_AppendScanLog_RejectsOrphanOrder(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders`).
		WithArgs(int64(999)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))
	_, err := NewPGStore(mock, stubExists{}).AppendScanLog(context.Background(),
		ScanLog{OrderID: 999, WorkerID: 1, WorkerName: "张", TagID: 1, Result: "MATCH"})
	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("err=%v, want ErrOrderNotFound", err)
	}
}

// GetTicketItemByNo 工单详情读模型:同 ListTicketItems 联表语义,按 ticketNo 寻址单行。
// 该接口是师傅端工单详情接口(GET /tickets/:ticketNo)的工单头数据源,必须返回
// 客户姓名/手机/地址/产品/完成时间等联表字段,详见 OpenAPI TicketDetail。
func TestPGStore_GetTicketItemByNo(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// 19 columns matching the updated SELECT in GetTicketItemByNo.
	cols := []string{
		"id", "ticket_no", "order_id", "worker_id", "status",
		"o.status", "c.name", "c.phone", "po.name",
		"addr", "stage", "finished_at",
		"splitter_port", "pre_bind_tag", "schedule_slot",
		"cmp.type", "cmp.created_at", "cmp.remote_diagnosis", "cmp.sla_deadline",
	}
	mock.ExpectQuery(`FROM dispatch_tickets`).
		WithArgs("TIC-1").
		WillReturnRows(mock.NewRows(cols).
			AddRow(
				int64(1), "TIC-1", int64(7), int64(2), "DOING",
				"DONE", "王先生", "13800001234", "1000M 极速宽带",
				"望京X · 3栋501", int8(12), "2026-08-21 16:30",
				"SPL-01-01", "", "",
				"SINGLE_OUTAGE", "2026-08-21 10:00", "光功率过低", "2026-08-21 14:00",
			))

	s := NewPGStore(mock, stubExists{})
	it, err := s.GetTicketItemByNo(context.Background(), "TIC-1")
	if err != nil {
		t.Fatalf("GetTicketItemByNo: %v", err)
	}
	if it.CustomerName != "王先生" || it.CustomerPhone != "13800001234" {
		t.Fatalf("customer fields wrong: %+v", it)
	}
	if it.OfferName != "1000M 极速宽带" || it.FinishedAt != "2026-08-21 16:30" {
		t.Fatalf("offer/finishedAt wrong: %+v", it)
	}
	if it.Address != "望京X · 3栋501" || it.Stage != 12 {
		t.Fatalf("addr/stage wrong: %+v", it)
	}
	if it.SplitterPort != "SPL-01-01" {
		t.Fatalf("splitterPort wrong: %q", it.SplitterPort)
	}
	if it.FaultTypeLabel != "单户断网（紧急 SLA ≤4h）" {
		t.Fatalf("faultTypeLabel wrong: %q", it.FaultTypeLabel)
	}
	if it.RemoteDiagnosis != "光功率过低" {
		t.Fatalf("remoteDiagnosis wrong: %q", it.RemoteDiagnosis)
	}
	if it.ReportedAt != "2026-08-21 10:00" {
		t.Fatalf("reportedAt wrong: %q", it.ReportedAt)
	}
	// SlaLeftMinutes depends on time.Now() vs sla_deadline, just check >=0 or ==0 for test.
	if it.SlaLeftMinutes < 0 {
		t.Fatalf("slaLeftMinutes negative: %d", it.SlaLeftMinutes)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// GetTicketItemByNo 未命中返回 ErrOrderNotFound(对齐 GetDispatchTicketByNo 行为)。
func TestPGStore_GetTicketItemByNo_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`FROM dispatch_tickets`).
		WithArgs("MISSING").
		WillReturnRows(mock.NewRows([]string{
			"id", "ticket_no", "order_id", "worker_id", "status",
			"o.status", "c.name", "c.phone", "po.name",
			"addr", "stage", "finished_at",
			"splitter_port", "pre_bind_tag", "schedule_slot",
			"cmp.type", "cmp.created_at", "cmp.remote_diagnosis", "cmp.sla_deadline",
		}))

	s := NewPGStore(mock, stubExists{})
	_, err = s.GetTicketItemByNo(context.Background(), "MISSING")
	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("want ErrOrderNotFound, got %v", err)
	}
}
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
