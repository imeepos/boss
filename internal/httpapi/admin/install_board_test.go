package adminapi

// 施工看板路由回归:GET /dispatch-tickets 全量返回工单 + menu:install-board 门禁。
// 修复背景:看板页原调 /dispatch_tickets,admin 端从未注册该路由,页面恒 404
// (docs/notes/adopted/2026-08-28-admin-pages-theme-fix.md 遗留项 1)。

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

func f64ptr(v float64) *float64 { return &v }

// TestBoardDispatchTickets 契约:看板列表全量返回(含已指派工单),携带到场打卡事实。
func TestBoardDispatchTickets(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	arrived := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	wo := &fakeDispatchOrder{tickets: []order.DispatchTicket{
		{TicketID: 1, TicketNo: "DT-1", Status: "PENDING"},
		{TicketID: 2, TicketNo: "DT-2", WorkerID: 5, WorkerName: "张师傅", Status: "DOING",
			ArrivedAt: &arrived, ArriveLat: f64ptr(14.5995), ArriveLng: f64ptr(120.9842)},
	}}
	r := newDispatchRouter(wo, &fakeOrderLedger{}, &fakeWorkerSvc{}, mgr)

	w := getJSON(t, r, "/api/admin/v1/dispatch-tickets", token)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []order.DispatchTicket `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// 看板需全量口径统计待派/装维/已打卡:不过滤 worker(区别 /dispatch/pool)。
	if body.Code != 0 || len(body.Data.Items) != 2 {
		t.Fatalf("code=%d items=%d, want 全量 2 条", body.Code, len(body.Data.Items))
	}
	got := body.Data.Items[1]
	if got.ArrivedAt == nil || got.ArriveLat == nil || got.ArriveLng == nil {
		t.Fatalf("到场打卡事实丢失: %+v", got)
	}
}

// TestBoardDispatchTicketsPerm 契约:menu:install-board 门禁;
// 刻意不挂 menu:dispatch(technician 持有,000168 收权语义)。
func TestBoardDispatchTicketsPerm(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{User: &fakeUser{permOk: false}, WorkOrder: &fakeDispatchOrder{}}, mgr)

	w := getJSON(t, r, "/api/admin/v1/dispatch-tickets", token)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", w.Code)
	}
}
