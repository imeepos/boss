package adminapi

// 开网订单流程推进端点(CLI 模拟 / 联调用):环节 2/3/4 人工推进 + 收费后自动段 5-8 + 激活后自动段 10-12。
// 验收口径(terms.md §1 12 环节):人工只做资源核查(2)、端口预占(3)、合同收费(4)与上门激活(10);
// 标签预绑定/建档/预下发/派单(5-8)与回调/GIS(11-12)由 Automation 自动推进。
// 全部 handler 实现见 order_workflow_handlers.go。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerOrderWorkflowRoutes 注册订单流程推进路由(承接 order.yaml workflow 段)。
func registerOrderWorkflowRoutes(g *gin.RouterGroup, a *app.Application) {
	wf := g.Group("/orders", requirePerm(a.User, "menu:order"))
	wf.GET("/:orderNo/check-preview", orderCheckPreviewHandler(a))
	wf.POST("/:orderNo/check-resource", orderCheckResourceHandler(a))
	wf.POST("/:orderNo/reserve", orderReserveHandler(a))
	wf.POST("/:orderNo/charge", orderChargeHandler(a))
	wf.POST("/:orderNo/cancel", orderCancelHandler(a))

	// 环节10 激活(上门扫码后,师傅上报装维结果):自动段 10-12(激活/回调/更新GIS)→ 订单 DONE。
	// 与扫码路由同门禁:任何已认证主体可调(worker 主体/账号),身份经 Subject 或 claims 识别。
	tic := g.Group("/tickets")
	tic.POST("/:ticketNo/activate", orderActivateHandler(a))
}
