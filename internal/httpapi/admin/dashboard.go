package adminapi

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/pkg/clock"
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
		"trend":           orderTrend(orders, c.Query("trendPeriod"), clock.Now()),
	}, nil
}

// dashboardStats 统计卡:今日订单/进行中工单/待处理告警/四码一致率。
func dashboardStats(orders []order.OrderListItem, tickets []order.DispatchTicket,
	alarms []device.Alarm, links []quadlink.QuadLink) []gin.H {
	today := countIf(len(orders), func(i int) bool { return sameDay(orders[i].CreatedAt, clock.Now()) })
	active := countIf(len(tickets), func(i int) bool { return tickets[i].Status == "DOING" })
	open := countIf(len(alarms), func(i int) bool { return alarms[i].Status == "OPEN" })
	rate := linkedRate(links)
	return []gin.H{
		{"key": "todayOrders", "label": "今日新增订单", "value": fmt.Sprint(today), "delta": "", "trend": "flat"},
		{"key": "activeTickets", "label": "进行中工单", "value": fmt.Sprint(active), "delta": "", "trend": "flat"},
		{"key": "pendingAlarms", "label": "待处理告警", "value": fmt.Sprint(open), "delta": "", "trend": "flat"},
		{"key": "assetConsistency", "label": "四码一致率", "value": rate, "delta": "", "trend": "flat"},
	}
}

// countIf 按下标谓词计数。
func countIf(n int, pred func(i int) bool) int {
	c := 0
	for i := 0; i < n; i++ {
		if pred(i) {
			c++
		}
	}
	return c
}

// linkedRate 四码一致率(LINKED 占比);无链路回"—"。
func linkedRate(links []quadlink.QuadLink) string {
	linked, total := 0, 0
	for _, l := range links {
		total++
		if l.Status == "LINKED" {
			linked++
		}
	}
	if total == 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", float64(linked)/float64(total)*100)
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
				"source": "告警中心", "time": al.CreatedAt.In(clock.Location()).Format("15:04"),
			})
		}
	}
	return items
}

// orderTrend 按周期聚合订单创建时间;年和全部按月,避免横轴点位过密。
func orderTrend(orders []order.OrderListItem, period string, now time.Time) gin.H {
	period = normalizeTrendPeriod(period)
	start, step, count, layout := trendWindow(period, orders, now)
	days := make([]string, count)
	for i := range days {
		days[i] = start.AddDate(0, 0, i*step).Format(layout)
	}
	return gin.H{"period": period, "days": days, "series": trendSeries(orders, days, start, step, period)}
}

func trendSeries(orders []order.OrderListItem, days []string, start time.Time, step int, period string) []gin.H {
	labels := map[string]string{
		"PENDING": "待核查", "RESERVED": "已预占", "INSTALLING": "装维中",
		"DONE": "已完成", "CANCELLED": "已取消",
	}
	statuses := []string{"PENDING", "RESERVED", "INSTALLING", "DONE", "CANCELLED"}
	series := make([]gin.H, 0, len(statuses))
	for _, status := range statuses {
		values := make([]int, len(days))
		for i := range days {
			point := start.AddDate(0, 0, i*step)
			for _, item := range orders {
				if item.Status == status && trendBucket(item.CreatedAt, point, step, period) {
					values[i]++
				}
			}
		}
		series = append(series, gin.H{"status": status, "statusLabel": labels[status], "values": values})
	}
	return series
}

func normalizeTrendPeriod(period string) string {
	switch period {
	case "week", "month", "quarter", "year", "all":
		return period
	default:
		return "week"
	}
}

func trendWindow(period string, orders []order.OrderListItem, now time.Time) (time.Time, int, int, string) {
	local := now.In(now.Location())
	switch period {
	case "month":
		return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, local.Location()), 1, local.Day(), "01-02"
	case "quarter":
		month := (int(local.Month())-1)/3*3 + 1
		start := time.Date(local.Year(), time.Month(month), 1, 0, 0, 0, 0, local.Location())
		return start, 1, int(local.Sub(start).Hours()/24) + 1, "01-02"
	case "year":
		return time.Date(local.Year(), 1, 1, 0, 0, 0, 0, local.Location()), 1, int(local.Month()), "2006-01"
	case "all":
		start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, local.Location())
		for _, o := range orders {
			created := o.CreatedAt.In(local.Location())
			if created.Before(start) {
				start = time.Date(created.Year(), created.Month(), 1, 0, 0, 0, 0, local.Location())
			}
		}
		count := (local.Year()-start.Year())*12 + int(local.Month()-start.Month()) + 1
		return start, 1, count, "2006-01"
	default:
		start := local.AddDate(0, 0, -((int(local.Weekday()) + 6) % 7))
		return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, local.Location()), 1, 7, "01-02"
	}
}

func trendBucket(created, point time.Time, step int, period string) bool {
	created = created.In(point.Location())
	if period == "all" || period == "year" {
		return created.Year() == point.Year() && created.Month() == point.Month()
	}
	next := point.AddDate(0, 0, step)
	return !created.Before(point) && created.Before(next)
}

// sameDay 同日判定(先归一到 b 的时区;pgx 回扫带进程时区而 clock.Now() 带
// 业务时区,直接比较会在业务时区 00:00-08:00 期间把当日订单算进前一天)。
func sameDay(a, b time.Time) bool {
	a = a.In(b.Location())
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
