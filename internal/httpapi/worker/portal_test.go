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
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

type fakePortalWorkerSvc struct {
	worker.WorkerService
}

func (f *fakePortalWorkerSvc) ListWorkers(context.Context, int64) ([]worker.Worker, error) {
	return []worker.Worker{{
		ID: 7, StaffNo: "WK-1007", Name: "张师傅", GroupID: 1,
		RegionID: 1, Phone: "13800001234", Status: 1, JoinedAt: time.Now(),
	}}, nil
}

func (f *fakePortalWorkerSvc) GetWorker(_ context.Context, id int64) (*worker.Worker, error) {
	if id != 7 {
		return nil, worker.ErrNotFound
	}
	return &worker.Worker{ID: 7, StaffNo: "WK-1007", Name: "张师傅", Phone: "13800001234", Status: 1}, nil
}

type fakePortalWorkOrder struct {
	order.WorkOrderService
	tickets  []order.DispatchTicket
	assigned int64
}

func (f *fakePortalWorkOrder) ListDispatchTickets(context.Context) ([]order.DispatchTicket, error) {
	return f.tickets, nil
}

func (f *fakePortalWorkOrder) GetDispatchTicketByNo(_ context.Context, no string) (*order.DispatchTicket, error) {
	for _, t := range f.tickets {
		if t.TicketNo == no {
			return &t, nil
		}
	}
	return nil, order.ErrOrderNotFound
}

func (f *fakePortalWorkOrder) AssignDispatchTicket(_ context.Context, no string, workerID int64, _ string) error {
	f.assigned = workerID
	return nil
}

type fakePortalOrder struct {
	order.OrderService
	activated int64
}

func (f *fakePortalOrder) Track(context.Context, int64) (*order.Order, []order.StageLog, error) {
	return &order.Order{ID: 1, OrderNo: "ORD-1", Stage: 9, Status: "INSTALLING"}, nil, nil
}

func (f *fakePortalOrder) ActivateUser(_ context.Context, id int64) error {
	f.activated = id
	return nil
}

func portalTestRouter(t *testing.T, fw *fakePortalWorkOrder, fo *fakePortalOrder) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv("BOSS_JWT_SECRET", "portal-test-secret")
	r := gin.New()
	a := &app.Application{
		Worker: &fakePortalWorkerSvc{}, WorkOrder: fw, Order: fo,
		Portal: portal.NewMemory(),
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
	// 抢单成功
	res = portalWorkerDo(r, "POST", "/api/worker/v1/hall/ORD-1/grab", "", token)
	if res["code"].(float64) != 0 || fw.assigned != 7 {
		t.Fatalf("grab failed: %v assigned=%d", res, fw.assigned)
	}
	// 我的工单列表(抢单后归属师傅 7)
	fw.tickets[0].WorkerID = 7
	fw.tickets[0].Status = "DOING"
	res = portalWorkerDo(r, "GET", "/api/worker/v1/tickets", "", token)
	items := res["data"].(map[string]any)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("tickets: %v", res)
	}
}
