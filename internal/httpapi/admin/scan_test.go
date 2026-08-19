package adminapi

// W5 扫码闭环 handler 测试:不一致拒(40920)、不扫码拦(42200)、MATCH 推进环节9。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

type fakeQuadlink struct {
	quadlink.QuadLinkService
	verifyRes string
	verifyErr error
	unbindErr error
	verified  *quadlink.ScanReq
	unbound   [2]any // orderID, epc
}

func (f *fakeQuadlink) VerifyScan(_ context.Context, r quadlink.ScanReq) (string, error) {
	f.verified = &r
	return f.verifyRes, f.verifyErr
}
func (f *fakeQuadlink) UnbindRequireScan(_ context.Context, orderID int64, epc string) error {
	f.unbound = [2]any{orderID, epc}
	return f.unbindErr
}

type fakeWorkOrder struct {
	order.WorkOrderService
	ticket *order.DispatchTicket
}

func (f *fakeWorkOrder) GetDispatchTicketByNo(_ context.Context, no string) (*order.DispatchTicket, error) {
	if f.ticket != nil && f.ticket.TicketNo == no {
		return f.ticket, nil
	}
	return nil, order.ErrOrderNotFound
}
func (f *fakeWorkOrder) ListScanLogs(context.Context, int64) ([]order.ScanLog, error) {
	return nil, nil
}

type fakeOrderScan struct {
	order.OrderService
	scanBound int64
}

func (f *fakeOrderScan) ScanBind(_ context.Context, id int64) error { f.scanBound = id; return nil }

func scanRouter(fq *fakeQuadlink, fw *fakeWorkOrder, fo *fakeOrderScan) *gin.Engine {
	r := gin.New()
	mgr := auth.NewManager("test-secret", time.Hour)
	a := &app.Application{User: &fakeUser{permOk: true}, QuadLink: fq, WorkOrder: fw, Order: fo}
	g := r.Group("/api/admin/v1", middleware.Authn(mgr, auth.AudAdmin))
	registerScanRoutes(g, a)
	return r
}

func scanDo(r *gin.Engine, method, path, body string) (*httptest.ResponseRecorder, string) {
	tok, _ := auth.NewManager("test-secret", time.Hour).Sign(auth.AudAdmin, 2, "张师傅", "technician")
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w, w.Body.String()
}

func ticket() *order.DispatchTicket {
	return &order.DispatchTicket{TicketID: 1, TicketNo: "TIC-1", OrderID: 7, WorkerID: 2, WorkerName: "张师傅"}
}

func TestScanBindHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("MATCH → 推进环节9", func(t *testing.T) {
		fq, fo := &fakeQuadlink{verifyRes: "MATCH"}, &fakeOrderScan{}
		_, body := scanDo(scanRouter(fq, &fakeWorkOrder{ticket: ticket()}, fo), http.MethodPost,
			"/api/admin/v1/tickets/TIC-1/scan-bind", `{"epc":"EPC-OK"}`)
		if !strings.Contains(body, `"MATCH"`) {
			t.Fatalf("body=%s", body)
		}
		if fo.scanBound != 7 {
			t.Fatalf("scanBound=%d, want 7", fo.scanBound)
		}
		if fq.verified.WorkerID != 2 { // 师傅身份取自 JWT
			t.Fatalf("verified=%+v", fq.verified)
		}
	})

	t.Run("MISMATCH → 40920 且不推进", func(t *testing.T) {
		fq, fo := &fakeQuadlink{verifyRes: "MISMATCH", verifyErr: quadlink.ErrScanMismatch}, &fakeOrderScan{}
		_, body := scanDo(scanRouter(fq, &fakeWorkOrder{ticket: ticket()}, fo), http.MethodPost,
			"/api/admin/v1/tickets/TIC-1/scan-bind", `{"epc":"EPC-BAD"}`)
		if !strings.Contains(body, `"code":40920`) {
			t.Fatalf("body=%s", body)
		}
		if fo.scanBound != 0 {
			t.Fatalf("scanBound=%d, want 0", fo.scanBound)
		}
	})

	t.Run("拆机不扫码 → 拦截", func(t *testing.T) {
		fq := &fakeQuadlink{unbindErr: quadlink.ErrScanRequired}
		_, body := scanDo(scanRouter(fq, &fakeWorkOrder{ticket: ticket()}, &fakeOrderScan{}), http.MethodPost,
			"/api/admin/v1/tickets/TIC-1/dismantle/scan", `{"epc":""}`)
		if !strings.Contains(body, `"code":42200`) {
			t.Fatalf("body=%s", body)
		}
	})

	t.Run("工单不存在 → 40400", func(t *testing.T) {
		_, body := scanDo(scanRouter(&fakeQuadlink{}, &fakeWorkOrder{}, &fakeOrderScan{}), http.MethodPost,
			"/api/admin/v1/tickets/TIC-X/scan-bind", `{"epc":"EPC"}`)
		if !strings.Contains(body, `"code":40400`) {
			t.Fatalf("body=%s", body)
		}
	})
}
