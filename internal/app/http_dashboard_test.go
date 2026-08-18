package app

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// dashboard 各域只读桩(嵌入接口,仅覆写用到的列表方法)。
type fakeDashOrder struct {
	order.OrderService
	list []order.OrderListItem
}

func (f *fakeDashOrder) List(context.Context, order.OrderQuery) ([]order.OrderListItem, error) {
	return f.list, nil
}

type fakeDashAlarm struct {
	device.AlarmService
	alarms []device.Alarm
}

func (f *fakeDashAlarm) ListAlarms(context.Context, int64) ([]device.Alarm, error) {
	return f.alarms, nil
}

type fakeDashQuadlink struct {
	quadlink.QuadLinkService
	links []quadlink.QuadLink
}

func (f *fakeDashQuadlink) ListLinks(context.Context) ([]quadlink.QuadLink, error) {
	return f.links, nil
}

// TestDashboard 契约:统计卡/订单状态分布/待办/近7日趋势均来自真实在库数据。
func TestDashboard(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	now := time.Now()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, &Application{
		User: &fakeUser{permOk: true},
		Order: &fakeDashOrder{list: []order.OrderListItem{
			{OrderNo: "ORD-1", Status: "PENDING", CreatedAt: now},
			{OrderNo: "ORD-2", Status: "DONE", CreatedAt: now.AddDate(0, 0, -1)},
		}},
		WorkOrder: &fakeDispatchOrder{tickets: []order.DispatchTicket{
			{TicketNo: "TK-1", WorkerID: 5, Status: "DOING"},
			{TicketNo: "TK-2", Status: "PENDING"},
		}},
		Alarm: &fakeDashAlarm{alarms: []device.Alarm{
			{ID: 1, AlarmNo: "ALM-1", Level: "CRITICAL", Content: "光功率越限", Status: "OPEN", CreatedAt: now},
		}},
		QuadLink: &fakeDashQuadlink{links: []quadlink.QuadLink{
			{Status: "LINKED"}, {Status: "CONFLICT"},
		}},
	}, mgr)

	w := getJSON(t, r, "/api/v1/dashboard", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Stats []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"stats"`
			OrderStatusDist []struct {
				Status string `json:"status"`
				Count  int    `json:"count"`
			} `json:"orderStatusDist"`
			Todos struct {
				Items []map[string]any `json:"items"`
			} `json:"todos"`
			Trend struct {
				Days   []string `json:"days"`
				Values []int    `json:"values"`
			} `json:"trend"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	stats := map[string]string{}
	for _, s := range body.Data.Stats {
		stats[s.Key] = s.Value
	}
	if stats["todayOrders"] != "1" || stats["activeTickets"] != "1" ||
		stats["pendingAlarms"] != "1" || stats["assetConsistency"] != "50.0%" {
		t.Fatalf("stats=%+v", stats)
	}
	dist := map[string]int{}
	for _, d := range body.Data.OrderStatusDist {
		dist[d.Status] = d.Count
	}
	if dist["PENDING"] != 1 || dist["DONE"] != 1 || len(body.Data.OrderStatusDist) != 4 {
		t.Fatalf("dist=%+v", dist)
	}
	// 待办 = 未派工单 TK-2 + OPEN 告警 ALM-1
	if len(body.Data.Todos.Items) != 2 {
		t.Fatalf("todos=%+v", body.Data.Todos.Items)
	}
	if len(body.Data.Trend.Days) != 7 || len(body.Data.Trend.Values) != 7 ||
		body.Data.Trend.Values[6] != 1 || body.Data.Trend.Values[5] != 1 {
		t.Fatalf("trend=%+v", body.Data.Trend)
	}
}
