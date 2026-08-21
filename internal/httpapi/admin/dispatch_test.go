package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeDispatchOrder 桩 order.WorkOrderService。
type fakeDispatchOrder struct {
	tickets  []order.DispatchTicket
	byNo     *order.DispatchTicket
	assigned *struct {
		ticketNo   string
		workerID   int64
		workerName string
	}
}

func (f *fakeDispatchOrder) ListDispatchTickets(context.Context) ([]order.DispatchTicket, error) {
	return f.tickets, nil
}

func (f *fakeDispatchOrder) ListTicketItems(context.Context) ([]order.TicketItem, error) {
	return nil, nil
}
func (f *fakeDispatchOrder) GetDispatchTicketByNo(_ context.Context, no string) (*order.DispatchTicket, error) {
	if f.byNo != nil {
		f.byNo.TicketNo = no
	}
	return f.byNo, nil
}
func (f *fakeDispatchOrder) AssignDispatchTicket(_ context.Context, no string, id int64, name string) error {
	f.assigned = &struct {
		ticketNo   string
		workerID   int64
		workerName string
	}{no, id, name}
	return nil
}
func (f *fakeDispatchOrder) AssignPendingDispatchTicket(_ context.Context, no string, id int64, name string) error {
	f.assigned = &struct {
		ticketNo   string
		workerID   int64
		workerName string
	}{no, id, name}
	return nil
}
func (f *fakeDispatchOrder) CreateDispatchTicket(context.Context, order.DispatchTicket) (int64, error) {
	return 0, nil
}
func (f *fakeDispatchOrder) ListComplaints(context.Context) ([]order.Complaint, error) {
	return nil, nil
}
func (f *fakeDispatchOrder) CreateComplaint(context.Context, order.Complaint) (int64, error) {
	return 0, nil
}
func (f *fakeDispatchOrder) CloseComplaint(context.Context, string) error {
	return nil
}
func (f *fakeDispatchOrder) ListScanLogs(context.Context, int64) ([]order.ScanLog, error) {
	return nil, nil
}
func (f *fakeDispatchOrder) AppendScanLog(context.Context, order.ScanLog) (int64, error) {
	return 0, nil
}

// fakeOrderLedger 桩 order.OrderLedgerService(仅改派台账落账)。
type fakeOrderLedger struct {
	transfers []order.DispatchTransfer
	appended  *order.DispatchTransfer
}

func (f *fakeOrderLedger) ListDismantles(context.Context) ([]order.Dismantle, error) {
	return nil, nil
}
func (f *fakeOrderLedger) CreateDismantle(context.Context, order.Dismantle) (int64, error) {
	return 0, nil
}
func (f *fakeOrderLedger) ListActivationCallbacks(context.Context) ([]order.ActivationCallback, error) {
	return nil, nil
}
func (f *fakeOrderLedger) AppendActivationCallback(context.Context, order.ActivationCallback) (int64, error) {
	return 0, nil
}
func (f *fakeOrderLedger) RetryActivationCallback(context.Context, int64) error {
	return nil
}
func (f *fakeOrderLedger) ListDispatchTransfers(context.Context, int64) ([]order.DispatchTransfer, error) {
	return f.transfers, nil
}
func (f *fakeOrderLedger) AppendDispatchTransfer(_ context.Context, t order.DispatchTransfer) (int64, error) {
	f.appended = &t
	return 1, nil
}

// fakeWorkerSvc 桩 worker.WorkerService。
type fakeWorkerSvc struct{ w *worker.Worker }

func (f *fakeWorkerSvc) ListGroups(context.Context) ([]worker.Group, error) { return nil, nil }
func (f *fakeWorkerSvc) CreateGroup(context.Context, worker.Group) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerSvc) ListWorkers(context.Context, int64) ([]worker.Worker, error) {
	return nil, nil
}
func (f *fakeWorkerSvc) CreateWorker(context.Context, worker.Worker) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerSvc) GetWorker(context.Context, int64) (*worker.Worker, error) {
	return f.w, nil
}

func newDispatchRouter(wo *fakeDispatchOrder, ol *fakeOrderLedger, ws *fakeWorkerSvc, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{
		User: &fakeUser{permOk: true}, WorkOrder: wo, OrderLedger: ol, Worker: ws,
	}, mgr)
	return r
}

// TestDispatchPool 契约:工单池仅含未指派工单。
func TestDispatchPool(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	wo := &fakeDispatchOrder{tickets: []order.DispatchTicket{
		{TicketNo: "TK-1", Status: "PENDING"},
		{TicketNo: "TK-2", WorkerID: 5, WorkerName: "张师傅", Status: "DOING"},
	}}
	r := newDispatchRouter(wo, &fakeOrderLedger{}, &fakeWorkerSvc{}, mgr)

	w := getJSON(t, r, "/api/admin/v1/dispatch/pool", authToken(t, mgr))
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []order.DispatchTicket `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 || len(body.Data.Items) != 1 || body.Data.Items[0].TicketNo != "TK-1" {
		t.Fatalf("body=%s", w.Body.String())
	}
}

// TestAssignTicket 契约:指派把候选师傅回填工单。
func TestAssignTicket(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	wo := &fakeDispatchOrder{}
	ws := &fakeWorkerSvc{w: &worker.Worker{ID: 5, Name: "张师傅"}}
	r := newDispatchRouter(wo, &fakeOrderLedger{}, ws, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/dispatch/pool/TK-1/assign", `{"masterId":5}`, authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if wo.assigned == nil || wo.assigned.ticketNo != "TK-1" ||
		wo.assigned.workerID != 5 || wo.assigned.workerName != "张师傅" {
		t.Fatalf("assigned=%+v", wo.assigned)
	}
}

// TestMyTickets 契约:按 workerId 过滤我的工单。
func TestMyTickets(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	wo := &fakeDispatchOrder{tickets: []order.DispatchTicket{
		{TicketNo: "TK-1", WorkerID: 5},
		{TicketNo: "TK-2", WorkerID: 6},
	}}
	r := newDispatchRouter(wo, &fakeOrderLedger{}, &fakeWorkerSvc{}, mgr)

	w := getJSON(t, r, "/api/admin/v1/dispatch/my-tickets?workerId=5", authToken(t, mgr))
	var body struct {
		Data struct {
			Items []order.DispatchTicket `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data.Items) != 1 || body.Data.Items[0].TicketNo != "TK-1" {
		t.Fatalf("items=%+v", body.Data.Items)
	}
}

// TestTransferTicket 契约:转派留痕台账并把工单指到新师傅。
func TestTransferTicket(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	wo := &fakeDispatchOrder{byNo: &order.DispatchTicket{
		TicketID: 3, TicketNo: "TK-3", WorkerID: 5, WorkerName: "张师傅",
	}}
	ol := &fakeOrderLedger{}
	ws := &fakeWorkerSvc{w: &worker.Worker{ID: 6, Name: "李师傅"}}
	r := newDispatchRouter(wo, ol, ws, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/dispatch/tickets/TK-3/transfer",
		`{"toMasterId":6,"reason":"跨区改派"}`, authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if ol.appended == nil || ol.appended.FromWorkerID != 5 || ol.appended.ToWorkerID != 6 ||
		ol.appended.Reason != "跨区改派" || ol.appended.TicketID != 3 {
		t.Fatalf("appended=%+v", ol.appended)
	}
	if wo.assigned == nil || wo.assigned.workerID != 6 || wo.assigned.workerName != "李师傅" {
		t.Fatalf("assigned=%+v", wo.assigned)
	}
}

// TestDispatchTransfers 契约:改派台账列表可查。
func TestDispatchTransfers(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	ol := &fakeOrderLedger{transfers: []order.DispatchTransfer{
		{ID: 1, TicketID: 3, FromWorkerID: 5, ToWorkerID: 6, Reason: "跨区改派"},
	}}
	r := newDispatchRouter(&fakeDispatchOrder{}, ol, &fakeWorkerSvc{}, mgr)

	w := getJSON(t, r, "/api/admin/v1/dispatch/transfers", authToken(t, mgr))
	var body struct {
		Data struct {
			Items []order.DispatchTransfer `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data.Items) != 1 || body.Data.Items[0].Reason != "跨区改派" {
		t.Fatalf("items=%+v", body.Data.Items)
	}
}
