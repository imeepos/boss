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
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
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
func (f *fakeDispatchOrder) GetTicketItemByNo(context.Context, string) (*order.TicketItem, error) {
	return nil, nil
}
func (f *fakeDispatchOrder) GetDispatchTicketByNo(_ context.Context, no string) (*order.DispatchTicket, error) {
	if f.byNo != nil {
		f.byNo.TicketNo = no
	}
	return f.byNo, nil
}
func (f *fakeDispatchOrder) GetDispatchTicketByOrder(_ context.Context, id int64) (*order.DispatchTicket, error) {
	for i := range f.tickets {
		if f.tickets[i].OrderID == id {
			return &f.tickets[i], nil
		}
	}
	return nil, nil
}
func (f *fakeDispatchOrder) AssignDispatchTicket(_ context.Context, no string, id int64, name string, _ ...order.AssignOpt) error {
	f.assigned = &struct {
		ticketNo   string
		workerID   int64
		workerName string
	}{no, id, name}
	return nil
}
func (f *fakeDispatchOrder) AssignPendingDispatchTicket(_ context.Context, no string, id int64, name string, _ ...order.AssignOpt) error {
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
func (f *fakeDispatchOrder) ClaimDispatchTicket(context.Context, string, int64, string) error {
	return nil
}
func (f *fakeDispatchOrder) UpdateScheduleSlot(context.Context, string, string) error {
	return nil
}
func (f *fakeDispatchOrder) ListComplaints(context.Context) ([]order.Complaint, error) {
	return nil, nil
}
func (f *fakeDispatchOrder) ListComplaintsByCustomerPaged(context.Context, int64, int, int) ([]order.Complaint, bool, error) {
	return nil, false, nil
}
func (f *fakeDispatchOrder) GetComplaintByNoAndCustomer(_ context.Context, no string, _ int64) (*order.Complaint, error) {
	if f.byNo == nil {
		return nil, order.ErrOrderNotFound
	}
	c := order.Complaint{TicketNo: no}
	return &c, nil
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
func (f *fakeDispatchOrder) SubmitInstallLog(context.Context, order.InstallLog) (int64, error) {
	return 0, nil
}
func (f *fakeDispatchOrder) MarkArrived(context.Context, string, order.ArriveInput, int64) error {
	return nil
}
func (f *fakeDispatchOrder) ListInstallLogs(context.Context, int64) ([]order.InstallLog, error) {
	return nil, nil
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
func (f *fakeWorkerSvc) ListWorkers(context.Context, int64, string) ([]worker.Worker, error) {
	return nil, nil
}
func (f *fakeWorkerSvc) CreateWorker(context.Context, worker.Worker) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerSvc) CreateWorkerWithPassword(context.Context, worker.Worker, string) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerSvc) SetPassword(context.Context, int64, string) error { return nil }
func (f *fakeWorkerSvc) VerifyPassword(context.Context, int64, string) (bool, error) {
	return false, nil
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
	ws := &fakeWorkerSvc{w: &worker.Worker{ID: 5, Name: "张师傅", Status: 1}}
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

// TestAssignTicketRejectsInactiveWorker 回归:离职/停用师傅不可被指派(修复前无状态校验)。
func TestAssignTicketRejectsInactiveWorker(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	wo := &fakeDispatchOrder{}
	ws := &fakeWorkerSvc{w: &worker.Worker{ID: 5, Name: "离职师傅", Status: 0}}
	r := newDispatchRouter(wo, &fakeOrderLedger{}, ws, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/dispatch/pool/TK-1/assign", `{"masterId":5}`, authToken(t, mgr))
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code == 0 || wo.assigned != nil {
		t.Fatalf("code=%d assigned=%+v, want rejected", body.Code, wo.assigned)
	}
}

// TestTransferTicketRejects 回归:终态工单不可转派;目标即当前师傅拒绝。
func TestTransferTicketRejects(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	done := &fakeDispatchOrder{byNo: &order.DispatchTicket{
		TicketID: 3, TicketNo: "TK-D", WorkerID: 5, Status: "DONE",
	}}
	ws := &fakeWorkerSvc{w: &worker.Worker{ID: 6, Name: "李师傅", Status: 1}}
	r := newDispatchRouter(done, &fakeOrderLedger{}, ws, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/dispatch/tickets/TK-D/transfer",
		`{"toMasterId":6,"reason":"终态"}`, authToken(t, mgr))
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code == 0 {
		t.Fatalf("terminal ticket transfer should be rejected, body=%s", w.Body.String())
	}

	self := &fakeDispatchOrder{byNo: &order.DispatchTicket{
		TicketID: 3, TicketNo: "TK-S", WorkerID: 6, Status: "PENDING",
	}}
	r2 := newDispatchRouter(self, &fakeOrderLedger{},
		&fakeWorkerSvc{w: &worker.Worker{ID: 6, Name: "李师傅", Status: 1}}, mgr)
	w2 := postBodyAuth(t, r2, "/api/admin/v1/dispatch/tickets/TK-S/transfer",
		`{"toMasterId":6,"reason":"自转"}`, authToken(t, mgr))
	var body2 struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &body2); err != nil {
		t.Fatal(err)
	}
	if body2.Code == 0 || self.assigned != nil {
		t.Fatalf("self-transfer should be rejected, code=%d assigned=%+v", body2.Code, self.assigned)
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
	ws := &fakeWorkerSvc{w: &worker.Worker{ID: 6, Name: "李师傅", Status: 1}}
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

// TestAssignTicketRegionMismatchForce 回归:跨区指派默认 40900 提醒,force 确认后放行。
func TestAssignTicketRegionMismatchForce(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	wo := &fakeDispatchOrder{byNo: &order.DispatchTicket{
		TicketNo: "TK-X", RegionID: 2, RegionName: "南区", Status: "PENDING",
	}}
	ws := &fakeWorkerSvc{w: &worker.Worker{ID: 5, Name: "张师傅", Status: 1, RegionID: 1}}
	r := newDispatchRouter(wo, &fakeOrderLedger{}, ws, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/dispatch/pool/TK-X/assign",
		`{"masterId":5}`, authToken(t, mgr))
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != int(apitypes.CodeConflict) || wo.assigned != nil {
		t.Fatalf("cross-region assign should 40900 and not assign: code=%d assigned=%+v", body.Code, wo.assigned)
	}

	w = postBodyAuth(t, r, "/api/admin/v1/dispatch/pool/TK-X/assign",
		`{"masterId":5,"force":true}`, authToken(t, mgr))
	if w.Code != http.StatusOK || wo.assigned == nil || wo.assigned.workerID != 5 {
		t.Fatalf("forced assign should pass: status=%d assigned=%+v body=%s", w.Code, wo.assigned, w.Body.String())
	}
}

// TestTransferTicketRegionMismatchForce 回归:跨区转派默认 40900 提醒,force 确认后放行。
func TestTransferTicketRegionMismatchForce(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	wo := &fakeDispatchOrder{byNo: &order.DispatchTicket{
		TicketID: 3, TicketNo: "TK-X", WorkerID: 5, RegionID: 2, RegionName: "南区", Status: "DOING",
	}}
	ol := &fakeOrderLedger{}
	ws := &fakeWorkerSvc{w: &worker.Worker{ID: 6, Name: "李师傅", Status: 1, RegionID: 1}}
	r := newDispatchRouter(wo, ol, ws, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/dispatch/tickets/TK-X/transfer",
		`{"toMasterId":6,"reason":"跨区"}`, authToken(t, mgr))
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != int(apitypes.CodeConflict) || wo.assigned != nil {
		t.Fatalf("cross-region transfer should 40900: code=%d assigned=%+v", body.Code, wo.assigned)
	}

	w = postBodyAuth(t, r, "/api/admin/v1/dispatch/tickets/TK-X/transfer",
		`{"toMasterId":6,"reason":"跨区","force":true}`, authToken(t, mgr))
	if w.Code != http.StatusOK || wo.assigned == nil || wo.assigned.workerID != 6 {
		t.Fatalf("forced transfer should pass: status=%d body=%s", w.Code, w.Body.String())
	}
}

// newTicketRouter 工单寻址路由(activate/transfer)测试装配:Automation 用 fakeOrder 兜底。
func newTicketRouter(wo *fakeDispatchOrder, u *fakeUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{
		User: u, WorkOrder: wo, OrderLedger: &fakeOrderLedger{}, Worker: &fakeWorkerSvc{},
		Automation: app.NewAutomation(&fakeOrder{}, nil),
	}, auth.NewManager("s", time.Hour))
	return r
}

// TestTicketScopedWriteDeniedAsNotFound 契约:数据范围外的工单推进/转派与不存在
// 同响应不可区分(2026-08-28 battle oracle;activate/transfer 曾无任何范围守卫)。
func TestTicketScopedWriteDeniedAsNotFound(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	// 区域越子树:工单 region_id=8(fakeUser.GetRegion 恒返 root.test),scope 区域为 visayas
	wo := &fakeDispatchOrder{byNo: &order.DispatchTicket{TicketID: 7, LegalEntityID: 1, RegionID: 8, Status: "DOING"}}
	u := &fakeUser{permOk: true, dataScope: user.DataScope{LegalEntityID: 1, RegionScope: "root.visayas"}}
	r := newTicketRouter(wo, u)

	wOut := postJSONAuth(t, r, "/api/admin/v1/tickets/TK-1/activate", "{}", authToken(t, mgr))
	wMiss := postJSONAuth(t, r, "/api/admin/v1/tickets/TK-404/activate", "{}", authToken(t, mgr))
	if wOut.Code != wMiss.Code || wOut.Body.String() != wMiss.Body.String() {
		t.Fatalf("越权推进与不存在可区分: out=%d %s miss=%d %s",
			wOut.Code, wOut.Body.String(), wMiss.Code, wMiss.Body.String())
	}

	// 实体不符:转派同样拦截
	wo2 := &fakeDispatchOrder{byNo: &order.DispatchTicket{TicketID: 7, LegalEntityID: 2, RegionID: 8, Status: "DOING"}}
	u2 := &fakeUser{permOk: true, dataScope: user.DataScope{LegalEntityID: 1, RegionScope: "root.test"}}
	wX := postJSONAuth(t, newTicketRouter(wo2, u2), "/api/admin/v1/dispatch/tickets/TK-1/transfer",
		`{"toMasterId":5,"reason":"e2e"}`, authToken(t, mgr))
	if wX.Code != http.StatusOK || wX.Body.String() == "" {
		t.Fatalf("transfer status=%d", wX.Code)
	}
	var body struct {
		Code apitypes.Code `json:"code"`
		Msg  string        `json:"msg"`
	}
	if err := json.Unmarshal(wX.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != apitypes.CodeNotFound {
		t.Fatalf("实体不符未被范围守卫拦截: %+v", body)
	}

	// 范围内放行:守卫通过后到达既有终态校验(ticket already DONE)
	wo3 := &fakeDispatchOrder{byNo: &order.DispatchTicket{TicketID: 7, LegalEntityID: 1, RegionID: 8, Status: "DONE"}}
	wIn := postJSONAuth(t, newTicketRouter(wo3, u2), "/api/admin/v1/dispatch/tickets/TK-1/transfer",
		`{"toMasterId":5,"reason":"e2e"}`, authToken(t, mgr))
	var bodyIn struct {
		Code apitypes.Code `json:"code"`
		Data struct {
			Error string `json:"error"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wIn.Body.Bytes(), &bodyIn); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if bodyIn.Code != apitypes.CodeInvalidParam || bodyIn.Data.Error != "ticket already DONE" {
		t.Fatalf("范围内工单被误拦: %+v", bodyIn)
	}
}

// TestActivateRequiresDispatchPerm 契约:activate 须持 menu:dispatch(2026-08-28 审计补门禁);
// 数据范围由 requireTicketInScope 另行把守,此处断言功能权限层。
func TestActivateRequiresDispatchPerm(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	wo := &fakeDispatchOrder{byNo: &order.DispatchTicket{TicketID: 7, Status: "DOING"}}
	r := newTicketRouter(wo, &fakeUser{permOk: false})
	w := postJSONAuth(t, r, "/api/admin/v1/tickets/TK-1/activate", "{}", authToken(t, mgr))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
