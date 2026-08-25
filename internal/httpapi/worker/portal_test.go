package workerapi

// W 师傅端门户测试:workerJWT 签发/校验、登录、鉴权门禁、工单列表/抢单。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

type fakePortalWorkerSvc struct {
	worker.WorkerService
	region int64
}

func (f *fakePortalWorkerSvc) ListWorkers(context.Context, int64, string) ([]worker.Worker, error) {
	return []worker.Worker{{
		ID: 7, StaffNo: "WK-1007", Name: "张师傅", GroupID: 1,
		RegionID: 1, Phone: "13800001234", Status: 1, JoinedAt: time.Now(),
	}}, nil
}

func (f *fakePortalWorkerSvc) GetWorker(_ context.Context, id int64) (*worker.Worker, error) {
	if id != 7 {
		return nil, worker.ErrNotFound
	}
	return &worker.Worker{ID: 7, StaffNo: "WK-1007", Name: "张师傅", Phone: "13800001234",
		Status: 1, RegionID: f.region}, nil
}

// fakePortalLedger 桩接单设置:未配置按 ErrNotFound(=默认在线全类型)。
type fakePortalLedger struct {
	worker.WorkerLedgerService
	settings map[int64]*worker.Settings
}

func (f *fakePortalLedger) GetSettings(_ context.Context, id int64) (*worker.Settings, error) {
	if s, ok := f.settings[id]; ok {
		return s, nil
	}
	return nil, worker.ErrNotFound
}

type fakePortalWorkOrder struct {
	order.WorkOrderService
	tickets  []order.DispatchTicket
	item     *order.TicketItem
	assigned int64
}

func (f *fakePortalWorkOrder) ListDispatchTickets(context.Context) ([]order.DispatchTicket, error) {
	return f.tickets, nil
}

func (f *fakePortalWorkOrder) ListTicketItems(context.Context) ([]order.TicketItem, error) {
	out := make([]order.TicketItem, 0, len(f.tickets))
	for _, t := range f.tickets {
		out = append(out, order.TicketItem{
			TicketID: t.TicketID, TicketNo: t.TicketNo, OrderID: t.OrderID,
			WorkerID: t.WorkerID, Status: t.Status,
			CustomerName: "测试客户", Address: "测试路 1 号", Stage: 9,
		})
	}
	return out, nil
}

func (f *fakePortalWorkOrder) GetDispatchTicketByNo(_ context.Context, no string) (*order.DispatchTicket, error) {
	for _, t := range f.tickets {
		if t.TicketNo == no {
			return &t, nil
		}
	}
	return nil, order.ErrOrderNotFound
}

func (f *fakePortalWorkOrder) AssignDispatchTicket(_ context.Context, no string, workerID int64, _ string, _ ...order.AssignOpt) error {
	f.assigned = workerID
	return nil
}

func (f *fakePortalWorkOrder) AssignPendingDispatchTicket(_ context.Context, no string, workerID int64, _ string, _ ...order.AssignOpt) error {
	for i := range f.tickets {
		if f.tickets[i].TicketNo == no {
			f.tickets[i].WorkerID = workerID
			f.tickets[i].Status = "DOING"
		}
	}
	f.assigned = workerID
	return nil
}

func (f *fakePortalWorkOrder) ClaimDispatchTicket(_ context.Context, no string, workerID int64, name string) error {
	for i := range f.tickets {
		if f.tickets[i].TicketNo == no {
			if f.tickets[i].Status != "PENDING" {
				return order.ErrOrderNotFound
			}
			f.tickets[i].WorkerID = workerID
			f.tickets[i].WorkerName = name
			f.tickets[i].Status = "DOING"
		}
	}
	f.assigned = workerID
	return nil
}

func (f *fakePortalWorkOrder) UpdateScheduleSlot(_ context.Context, _ string, _ string) error {
	return nil
}

func (f *fakePortalWorkOrder) GetTicketItemByNo(_ context.Context, _ string) (*order.TicketItem, error) {
	if f.item != nil {
		return f.item, nil
	}
	return &order.TicketItem{}, nil
}

type fakePortalOrder struct {
	order.OrderService
	activated  int64
	rolledBack int64
}

// fakePortalQuad 四码桩:详情页 quad 视图按未绑定兜底。
type fakePortalQuad struct {
	quadlink.QuadLinkService
}

func (f *fakePortalQuad) GetByAddress(context.Context, int64) (*quadlink.QuadLink, error) {
	return nil, quadlink.ErrNotFound
}

func (f *fakePortalOrder) Track(context.Context, int64) (*order.Order, []order.StageLog, error) {
	return &order.Order{ID: 1, OrderNo: "ORD-1", Stage: 9, Status: "INSTALLING"}, nil, nil
}

func (f *fakePortalOrder) ActivateUser(_ context.Context, id int64) error {
	f.activated = id
	return nil
}

func (f *fakePortalOrder) NotifyActivation(_ context.Context, _ int64) error { return nil }
func (f *fakePortalOrder) UpdateMap(_ context.Context, _ int64) error        { return nil }

func (f *fakePortalOrder) RollbackStage(_ context.Context, id int64) error {
	f.rolledBack = id
	return nil
}

func portalTestRouter(t *testing.T, fw *fakePortalWorkOrder, fo *fakePortalOrder) *gin.Engine {
	return portalTestRouterWith(t, fw, fo, &fakePortalWorkerSvc{}, nil)
}

// portalTestRouterWith 可注入区域/接单设置的完整装配。
func portalTestRouterWith(t *testing.T, fw *fakePortalWorkOrder, fo *fakePortalOrder,
	ws *fakePortalWorkerSvc, wl worker.WorkerLedgerService) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv("BOSS_JWT_SECRET", "portal-test-secret")
	r := gin.New()
	a := &app.Application{
		Worker: ws, WorkOrder: fw, Order: fo, WorkerLedger: wl,
		Portal:     portal.NewMemory(),
		QuadLink:   &fakePortalQuad{},
		Automation: app.NewAutomation(fo, nil),
	}
	Register(r, a, newWorkerJWTManager())
	return r
}

func portalWorkerDo(r *gin.Engine, method, path, body, token string) map[string]any {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	out["_status"] = w.Code
	return out
}

func TestWorkerTokenRoundTripAndAdminIsolation(t *testing.T) {
	t.Setenv("BOSS_JWT_SECRET", "portal-test-secret")
	tok, err := signWorkerToken(7, "张师傅")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	claims, err := verifyWorkerToken(tok)
	if err != nil || claims.WorkerID != 7 {
		t.Fatalf("verify: %v %+v", err, claims)
	}
	// admin 签发的 token(issuer=boss)不得通过师傅端校验
	if _, err := verifyWorkerToken(mustAdminToken(t)); err == nil {
		t.Fatal("admin token must not pass worker verify")
	}
}

func mustAdminToken(t *testing.T) string {
	t.Helper()
	tok, err := newWorkerJWTManager().Sign(auth.AudAdmin, 1, "admin", "sysadmin") // 同密钥,issuer=boss
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestPortalLoginAndTickets(t *testing.T) {
	fw := &fakePortalWorkOrder{tickets: []order.DispatchTicket{
		{TicketID: 1, TicketNo: "ORD-1", OrderID: 1, WorkerID: 0, Status: "PENDING"},
	}}
	fo := &fakePortalOrder{}
	r := portalTestRouter(t, fw, fo)

	// 先发码(落 portal_sms_codes,内存替身固定 123456)
	res := portalWorkerDo(r, "POST", "/api/worker/v1/auth/sms-code",
		`{"phone":"13800001234"}`, "")
	if res["code"].(float64) != 0 {
		t.Fatalf("sms-code failed: %v", res)
	}
	// 错验证码拒
	res = portalWorkerDo(r, "POST", "/api/worker/v1/auth/login",
		`{"phone":"13800001234","mode":"sms","smsCode":"12"}`, "")
	if res["code"].(float64) == 0 {
		t.Fatalf("bad code should fail: %v", res)
	}
	// 正确登录
	res = portalWorkerDo(r, "POST", "/api/worker/v1/auth/login",
		`{"phone":"13800001234","mode":"sms","smsCode":"123456"}`, "")
	token, _ := res["data"].(map[string]any)["token"].(string)
	if token == "" {
		t.Fatalf("login failed: %v", res)
	}
	// 未带 token 访问被拒
	res = portalWorkerDo(r, "GET", "/api/worker/v1/tickets", "", "")
	if res["_status"] != http.StatusUnauthorized {
		t.Fatalf("anon should 401: %v", res)
	}
	// 抢单成功:fake 内直接流转 PENDING→DOING,不再手工改状态(回归:抢单不改状态=列表永远"待领取")
	res = portalWorkerDo(r, "POST", "/api/worker/v1/hall/ORD-1/grab", "", token)
	if res["code"].(float64) != 0 || fw.assigned != 7 {
		t.Fatalf("grab failed: %v assigned=%d", res, fw.assigned)
	}
	if fw.tickets[0].Status != "DOING" {
		t.Fatalf("grab should flip status to DOING, got %s", fw.tickets[0].Status)
	}
	// 我的工单列表(抢单后归属师傅 7)
	res = portalWorkerDo(r, "GET", "/api/worker/v1/tickets", "", token)
	items := res["data"].(map[string]any)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("tickets: %v", res)
	}
	it := items[0].(map[string]any)
	// 列表视图字段来自读模型联表,不再占位
	if it["address"] != "测试路 1 号" || it["customerName"] != "测试客户" {
		t.Fatalf("ticket view not enriched: %v", it)
	}
	if it["statusLabel"] != "进行中" || it["stage"].(float64) != 9 {
		t.Fatalf("statusLabel/stage wrong: %v", it)
	}
}

// 回归:领取(accept)后工单必须 PENDING→DOING,否则列表刷新仍"待领取",观感"点击无反应"。
func TestPortalAcceptFlipsStatus(t *testing.T) {
	t.Setenv("BOSS_JWT_SECRET", "portal-test-secret")
	tok, err := signWorkerToken(7, "张师傅")
	if err != nil {
		t.Fatal(err)
	}
	fw := &fakePortalWorkOrder{tickets: []order.DispatchTicket{
		{TicketID: 1, TicketNo: "ORD-1", OrderID: 1, WorkerID: 7, Status: "PENDING"},
	}}
	r := portalTestRouter(t, fw, &fakePortalOrder{})
	res := portalWorkerDo(r, "POST", "/api/worker/v1/tickets/ORD-1/accept", "", tok)
	if res["code"].(float64) != 0 {
		t.Fatalf("accept failed: %v", res)
	}
	if fw.tickets[0].Status != "DOING" {
		t.Fatalf("accept should flip status to DOING, got %s", fw.tickets[0].Status)
	}
	// 重复领取:已非 PENDING,按状态无效拒绝
	res = portalWorkerDo(r, "POST", "/api/worker/v1/tickets/ORD-1/accept", "", tok)
	if res["code"].(float64) == 0 {
		t.Fatalf("re-accept should fail: %v", res)
	}
}

// portalGrabToken 测试专用 token(师傅 7);先定密钥再签发。
func portalGrabToken(t *testing.T) string {
	t.Helper()
	t.Setenv("BOSS_JWT_SECRET", "portal-test-secret")
	tok, err := signWorkerToken(7, "张师傅")
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// TestWorkerTicketDetailType 回归(ISSUE.md worker 详情字段缺口):
// 详情补 type/typeLabel;报障单(complaints 联表有值)推导 REPAIR,普通单 INSTALL。
func TestWorkerTicketDetailType(t *testing.T) {
	tok := portalGrabToken(t)
	fw := &fakePortalWorkOrder{tickets: []order.DispatchTicket{
		{TicketID: 1, TicketNo: "DT-1", OrderID: 1, WorkerID: 7, Status: "DOING"},
	}}
	r := portalTestRouter(t, fw, &fakePortalOrder{})

	res := portalWorkerDo(r, "GET", "/api/worker/v1/tickets/DT-1", "", tok)
	data, _ := res["data"].(map[string]any)
	if data["type"] != "INSTALL" || data["typeLabel"] != "新装" {
		t.Fatalf("install detail type=%v/%v", data["type"], data["typeLabel"])
	}

	fw.item = &order.TicketItem{ComplaintType: "NO_NET", FaultTypeLabel: "单户断网"}
	res = portalWorkerDo(r, "GET", "/api/worker/v1/tickets/DT-1", "", tok)
	data, _ = res["data"].(map[string]any)
	if data["type"] != "REPAIR" || data["typeLabel"] != "报障" {
		t.Fatalf("repair detail type=%v/%v", data["type"], data["typeLabel"])
	}
}

// TestWorkerRollback 回归(ISSUE.md rollback 仅审计不落库):
// 本人工单回退真实调用 RollbackStage;他人工单拒绝。
func TestWorkerRollback(t *testing.T) {
	tok := portalGrabToken(t)
	fw := &fakePortalWorkOrder{tickets: []order.DispatchTicket{
		{TicketID: 1, TicketNo: "DT-1", OrderID: 5, WorkerID: 7, Status: "DOING"},
		{TicketID: 2, TicketNo: "DT-2", OrderID: 6, WorkerID: 8, Status: "DOING"},
	}}
	fo := &fakePortalOrder{}
	r := portalTestRouter(t, fw, fo)

	res := portalWorkerDo(r, "POST", "/api/worker/v1/tickets/DT-1/rollback", "", tok)
	if res["code"].(float64) != 0 || fo.rolledBack != 5 {
		t.Fatalf("rollback res=%v rolledBack=%d, want ok/5", res, fo.rolledBack)
	}
	// 他人工单:拒绝且不动订单。
	res = portalWorkerDo(r, "POST", "/api/worker/v1/tickets/DT-2/rollback", "", tok)
	if res["code"].(float64) == 0 || fo.rolledBack != 5 {
		t.Fatalf("foreign ticket rollback should be rejected: %v", res)
	}
}

// TestGrabRejectsOutOfRegion 回归:任务池/抢单必须限制在师傅负责区域内。
func TestGrabRejectsOutOfRegion(t *testing.T) {
	tok := portalGrabToken(t)
	fw := &fakePortalWorkOrder{tickets: []order.DispatchTicket{
		{TicketID: 1, TicketNo: "ORD-X", OrderID: 1, WorkerID: 0, Status: "PENDING", RegionID: 2},
	}}
	r := portalTestRouterWith(t, fw, &fakePortalOrder{},
		&fakePortalWorkerSvc{region: 1}, &fakePortalLedger{})
	res := portalWorkerDo(r, "POST", "/api/worker/v1/hall/ORD-X/grab", "", tok)
	if res["code"].(float64) == 0 || fw.assigned != 0 {
		t.Fatalf("cross-region grab should be rejected: %v assigned=%d", res, fw.assigned)
	}
}

// TestHallFiltersByRegion 回归:任务池仅展示负责区域匹配的工单。
func TestHallFiltersByRegion(t *testing.T) {
	tok := portalGrabToken(t)
	fw := &fakePortalWorkOrder{tickets: []order.DispatchTicket{
		{TicketNo: "ORD-IN", WorkerID: 0, Status: "PENDING", RegionID: 1},
		{TicketNo: "ORD-OUT", WorkerID: 0, Status: "PENDING", RegionID: 2},
	}}
	r := portalTestRouterWith(t, fw, &fakePortalOrder{},
		&fakePortalWorkerSvc{region: 1}, &fakePortalLedger{})
	res := portalWorkerDo(r, "GET", "/api/worker/v1/hall", "", tok)
	items := res["data"].(map[string]any)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["ticketNo"] != "ORD-IN" {
		t.Fatalf("hall must only contain in-region tickets: %v", items)
	}
}

// TestGrabRejectsStoppedWorker 回归:停接单/类型不符的师傅不可抢单。
func TestGrabRejectsStoppedWorker(t *testing.T) {
	tok := portalGrabToken(t)
	fw := &fakePortalWorkOrder{tickets: []order.DispatchTicket{
		{TicketNo: "ORD-1", WorkerID: 0, Status: "PENDING", RegionID: 1},
	}}
	ledger := &fakePortalLedger{settings: map[int64]*worker.Settings{
		7: {WorkerID: 7, Accepting: false, RadiusKm: 5, AcceptTypes: ""},
	}}
	r := portalTestRouterWith(t, fw, &fakePortalOrder{}, &fakePortalWorkerSvc{region: 1}, ledger)
	res := portalWorkerDo(r, "POST", "/api/worker/v1/hall/ORD-1/grab", "", tok)
	if res["code"].(float64) == 0 || fw.assigned != 0 {
		t.Fatalf("stopped worker grab should be rejected: %v", res)
	}
}

// TestGrabRejectsAcceptTypeMismatch 回归:接单类型不含 INSTALL 时拒绝。
func TestGrabRejectsAcceptTypeMismatch(t *testing.T) {
	tok := portalGrabToken(t)
	fw := &fakePortalWorkOrder{tickets: []order.DispatchTicket{
		{TicketNo: "ORD-1", WorkerID: 0, Status: "PENDING", RegionID: 1},
	}}
	ledger := &fakePortalLedger{settings: map[int64]*worker.Settings{
		7: {WorkerID: 7, Accepting: true, RadiusKm: 5, AcceptTypes: "REPAIR"},
	}}
	r := portalTestRouterWith(t, fw, &fakePortalOrder{}, &fakePortalWorkerSvc{region: 1}, ledger)
	res := portalWorkerDo(r, "POST", "/api/worker/v1/hall/ORD-1/grab", "", tok)
	if res["code"].(float64) == 0 || fw.assigned != 0 {
		t.Fatalf("type-mismatched grab should be rejected: %v", res)
	}
}

// TestActivateEnforcesOwnership 回归:激活接口拒绝非本人工单。
func TestActivateEnforcesOwnership(t *testing.T) {
	tok := portalGrabToken(t)
	// 工单归属师傅 8(非当前师傅 7)
	fw := &fakePortalWorkOrder{tickets: []order.DispatchTicket{
		{TicketNo: "ORD-1", OrderID: 1, WorkerID: 8, Status: "DOING"},
	}}
	fo := &fakePortalOrder{}
	r := portalTestRouter(t, fw, fo)
	res := portalWorkerDo(r, "POST", "/api/worker/v1/tickets/ORD-1/activate", `{}`, tok)
	if res["code"].(float64) != 40300 {
		t.Fatalf("other worker's ticket activate should 40300, got %v", res)
	}
	// 归属正确时允许
	fw.tickets[0].WorkerID = 7
	res = portalWorkerDo(r, "POST", "/api/worker/v1/tickets/ORD-1/activate", `{}`, tok)
	if res["code"].(float64) != 0 {
		t.Fatalf("own ticket activate should succeed, got %v", res)
	}
}

// TestActivationStateReturnsLoid 回归:激活状态查询返回真实 LOID(当 Aaa 可用时);不可用时返回空。
func TestActivationStateReturnsLoid(t *testing.T) {
	tok := portalGrabToken(t)
	fw := &fakePortalWorkOrder{tickets: []order.DispatchTicket{
		{TicketNo: "ORD-1", OrderID: 1, WorkerID: 7, Status: "DOING"},
	}}
	fo := &fakePortalOrder{}
	r := portalTestRouter(t, fw, fo)
	res := portalWorkerDo(r, "GET", "/api/worker/v1/tickets/ORD-1/activation", "", tok)
	if res["_status"] != http.StatusOK {
		t.Fatalf("activation state should succeed, got %d: %v", res["_status"], res)
	}
	data := res["data"].(map[string]any)
	if data["loid"].(string) != "" {
		t.Fatalf("loid should be empty when Aaa not available, got %s", data["loid"])
	}
	if data["status"].(string) != "PENDING" {
		t.Fatalf("status should be PENDING at stage 9, got %s", data["status"])
	}
}
