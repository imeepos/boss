package adminapi

// 订单域路由注册(承接 api/openapi/admin/order.yaml)。
// 全部 handler 实现见 order_handlers.go;此处只保留扁平路由表 + 响应体/辅助。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
)

// stageNames 环节序号 → 中文名(terms.md §1)。
var stageNames = map[int8]string{
	1: "下单", 2: "资源核查", 3: "端口预占", 4: "合同收费", 5: "标签预绑定",
	6: "创建账号", 7: "预下发配置", 8: "派单", 9: "扫码绑定", 10: "激活", 11: "激活回调", 12: "更新GIS",
}

// orderListResp 订单列表项(承接 listOrders,ops 为可用操作)。
type orderListResp struct {
	ID         int64     `json:"id"`
	OrderNo    string    `json:"orderNo"`
	Customer   string    `json:"customer"`
	Product    string    `json:"product"`
	Address    string    `json:"address"`
	Stage      int8      `json:"stage"`
	StageLabel string    `json:"stageLabel"`
	Status     string    `json:"status"`
	Ops        []string  `json:"ops"`
	CreatedAt  time.Time `json:"createdAt"`
}

// timelineItem 订单详情时间轴项(承接 getOrder.timeline)。
type timelineItem struct {
	Stage      int8       `json:"stage"`
	Name       string     `json:"name"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	Duration   string     `json:"duration"`
	Retries    int        `json:"retries"`
	Result     string     `json:"result"`
}

// orderOps 按状态推导可用操作(跟踪/退订)。
func orderOps(status string) []string {
	ops := []string{"track"}
	if status != "DONE" && status != "CANCELLED" {
		ops = append(ops, "cancel")
	}
	return ops
}

// registerOrderRoutes 注册订单域路由(承接 api/openapi/admin/order.yaml)。
func registerOrderRoutes(g *gin.RouterGroup, a *app.Application) {
	ord := g.Group("/orders", requirePerm(a.User, "menu:order"))

	ord.GET("", orderListHandler(a))
	ord.POST("", orderSubmitHandler(a))
	ord.GET("/:orderNo", orderGetHandler(a))
}

// buildTimeline 环节日志 → 时间轴;耗时 = 本环节完成时间 - 上一环节完成时间(首环节为下单时间)。
func buildTimeline(o *order.Order, logs []order.StageLog) []timelineItem {
	out := make([]timelineItem, 0, len(logs))
	prev := o.CreatedAt
	for i, lg := range logs {
		item := timelineItem{Stage: lg.Stage, Name: stageNames[lg.Stage], Retries: lg.Retries, Result: lg.Result}
		if lg.FinishedAt != nil {
			item.FinishedAt = lg.FinishedAt
			base := prev
			if i > 0 && logs[i-1].FinishedAt != nil {
				base = *logs[i-1].FinishedAt
			}
			item.Duration = lg.FinishedAt.Sub(base).Round(time.Second).String()
		}
		out = append(out, item)
	}
	return out
}