package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/clock"
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
func (f *fakeDashQuadlink) PurgeOrphans(context.Context) (int64, error) { return 0, nil }

// TestDashboard 契约:统计卡/订单状态分布/待办/近7日趋势均来自真实在库数据。
func TestDashboard(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	if err := clock.Set("Asia/Manila"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = clock.Set("UTC")
		clock.SetFixed(time.Time{}) // 清除固定时刻,不泄漏给其他测试
	})
	// 钉死执行时刻:夹具与 handler 的 clock.Now() 完全一致,
	// 测试不随真实执行日期、宿主时区或日界附近的毫秒差漂移。
	manila := clock.Location()
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, manila)
	clock.SetFixed(now)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{
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

	w := getJSON(t, r, "/api/admin/v1/dashboard", authToken(t, mgr))
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
				Series []struct {
					Status string `json:"status"`
					Values []int  `json:"values"`
				} `json:"series"`
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
	if len(body.Data.Trend.Days) != 7 || len(body.Data.Trend.Series) != 5 {
		t.Fatalf("trend=%+v", body.Data.Trend)
	}
	// 固定时刻下的确定性窗口:2026-08-21(周五)所在周为 08-17(周一)起 7 天。
	if body.Data.Trend.Days[0] != "08-17" || body.Data.Trend.Days[6] != "08-23" {
		t.Fatalf("trend days=%v", body.Data.Trend.Days)
	}
	trend := map[string][]int{}
	for _, series := range body.Data.Trend.Series {
		trend[series.Status] = series.Values
	}
	if sumTrend(trend["PENDING"]) != 1 || sumTrend(trend["DONE"]) != 1 {
		t.Fatalf("trend series=%+v", trend)
	}
	if err := clock.Set("UTC"); err != nil {
		t.Fatal(err)
	}
	month := getJSON(t, r, "/api/admin/v1/dashboard?trendPeriod=month", authToken(t, mgr))
	if month.Code != http.StatusOK || !contains(month.Body.String(), `"period":"month"`) {
		t.Fatalf("month trend=%s", month.Body.String())
	}
}

func contains(body, fragment string) bool { return strings.Contains(body, fragment) }

func sumTrend(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

// TestSameDayTimezone 回归:DB 时间戳为 UTC,now 为本地时区时不得把当日订单算进前一天。
func TestSameDayTimezone(t *testing.T) {
	local := time.FixedZone("CST", 8*3600)            // 固定 +8,避免宿主机时区影响断言
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, local) // 本地 08-21 10:00
	utc := time.FixedZone("UTC", 0)
	cases := []struct {
		ts     time.Time
		want   bool
		descEv string
	}{
		{time.Date(2026, 8, 21, 2, 0, 0, 0, utc), true, "UTC 02:00 = 本地 10:00 当日"},
		{time.Date(2026, 8, 20, 20, 0, 0, 0, utc), true, "UTC 20:00 = 本地当日 04:00"},
		{time.Date(2026, 8, 20, 10, 0, 0, 0, utc), false, "UTC 10:00 = 本地前一日 18:00"},
		{time.Date(2026, 8, 21, 10, 0, 0, 0, local), true, "本地当日"},
	}
	for _, c := range cases {
		if got := sameDay(c.ts, now); got != c.want {
			t.Fatalf("%s: sameDay=%v want=%v", c.descEv, got, c.want)
		}
	}
}
