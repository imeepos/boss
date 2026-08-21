package adminapi

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerDashboardRoutes 注册工作台聚合路由(dashboard.yaml getDashboard)。
func registerDashboardRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/dashboard", requirePerm(a.User, "menu:dashboard"), func(c *gin.Context) {
		d, err := buildDashboard(a, c)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, d)
	})
}

// buildDashboard 聚合工作台数据:统计卡 + 订单状态分布 + 待办 + 近7日趋势。
func buildDashboard(a *app.Application, c *gin.Context) (gin.H, error) {
	ctx := c.Request.Context()
	orders, err := a.Order.List(ctx, order.OrderQuery{})
	if err != nil {
		return nil, err
	}
	tickets, err := a.WorkOrder.ListDispatchTickets(ctx)
	if err != nil {
		return nil, err
	}
	alarms, err := a.Alarm.ListAlarms(ctx, 0)
	if err != nil {
		return nil, err
	}
	links, err := a.QuadLink.ListLinks(ctx)
	if err != nil {
		return nil, err
	}
	return gin.H{
		"stats":           dashboardStats(orders, tickets, alarms, links),
		"orderStatusDist": orderStatusDist(orders),
		"todos":           gin.H{"items": dashboardTodos(alarms, tickets)},
		"trend":           weeklyTrend(orders, time.Now()),
	}, nil
}

// dashboardStats 统计卡:今日订单/进行中工单/待处理告警/四码一致率。
func dashboardStats(orders []order.OrderListItem, tickets []order.DispatchTicket,
	alarms []device.Alarm, links []quadlink.QuadLink) []gin.H {
	today, active, open := 0, 0, 0
	for _, o := range orders {
		if sameDay(o.CreatedAt, time.Now()) {
			today++
		}
	}
	for _, t := range tickets {
		if t.Status == "DOING" {
			active++
		}
	}
	for _, al := range alarms {
		if al.Status == "OPEN" {
			open++
		}
	}
	linked, total := 0, 0
	for _, l := range links {
		total++
		if l.Status == "LINKED" {
			linked++
		}
	}
	rate := "—"
	if total > 0 {
		rate = fmt.Sprintf("%.1f%%", float64(linked)/float64(total)*100)
	}
	return []gin.H{
		{"key": "todayOrders", "label": "今日新增订单", "value": fmt.Sprint(today), "delta": "", "trend": "flat"},
		{"key": "activeTickets", "label": "进行中工单", "value": fmt.Sprint(active), "delta": "", "trend": "flat"},
		{"key": "pendingAlarms", "label": "待处理告警", "value": fmt.Sprint(open), "delta": "", "trend": "flat"},
		{"key": "assetConsistency", "label": "四码一致率", "value": rate, "delta": "", "trend": "flat"},
	}
}

// orderStatusDist 订单状态分布(terms.md 状态枚举 + 占比)。
func orderStatusDist(orders []order.OrderListItem) []gin.H {
	labels := map[string]string{
		"PENDING": "待核查", "RESERVED": "已预占", "INSTALLING": "装维中", "DONE": "已完成",
	}
	counts := map[string]int{}
	for _, o := range orders {
		counts[o.Status]++
	}
	dist := make([]gin.H, 0, len(labels))
	for _, st := range []string{"PENDING", "RESERVED", "INSTALLING", "DONE"} {
		pct := "0%"
		if len(orders) > 0 {
			pct = fmt.Sprintf("%.0f%%", float64(counts[st])/float64(len(orders))*100)
		}
		dist = append(dist, gin.H{
			"status": st, "statusLabel": labels[st], "count": counts[st], "percent": pct,
		})
	}
	return dist
}

// dashboardTodos 待办:未派工单 + OPEN 告警(来源于真实在库记录)。
func dashboardTodos(alarms []device.Alarm, tickets []order.DispatchTicket) []gin.H {
	items := make([]gin.H, 0)
	for _, t := range tickets {
		if t.WorkerID == 0 && t.Status == "PENDING" {
			items = append(items, gin.H{
				"todoId": t.TicketID, "subject": t.TicketNo + " 待指派师傅",
				"source": "派单池", "time": "",
			})
		}
	}
	for _, al := range alarms {
		if al.Status == "OPEN" {
			items = append(items, gin.H{
				"todoId": al.ID, "subject": al.AlarmNo + " " + al.Content,
				"source": "告警中心", "time": al.CreatedAt.Local().Format("15:04"),
			})
		}
	}
	return items
}

// weeklyTrend 近7日下单趋势(按订单创建日聚合)。
func weeklyTrend(orders []order.OrderListItem, now time.Time) gin.H {
	days := make([]string, 7)
	counts := make([]int, 7)
	for i := 6; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		days[6-i] = day.Format("01-02")
		for _, o := range orders {
			if sameDay(o.CreatedAt, day) {
				counts[6-i]++
			}
		}
	}
	return gin.H{"days": days, "values": counts}
}

// sameDay 同日判定(先归一到 b 的时区:DB 时间戳按 UTC 扫描,time.Now() 为服务器本地时区,
// 直接比较会在本地 00:00-08:00 期间把当日订单算进前一天)。
func sameDay(a, b time.Time) bool {
	a = a.In(b.Location())
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
