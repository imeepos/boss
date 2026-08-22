package adminapi

// 订单域子表路由注册:报障投诉/拆机/激活回调(order.yaml 已声明、原未实现的 5 个端点)。
// 全部 handler 实现见 order_sub_handlers.go;此处只保留扁平路由表。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerOrderSubRoutes 注册订单子表路由(承接 api/openapi/admin/order.yaml)。
func registerOrderSubRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/complaints", requirePerm(a.User, "menu:complaint"), orderListComplaintsHandler(a))
	g.POST("/complaints/:ticketNo/close", requirePerm(a.User, "menu:complaint"), orderCloseComplaintHandler(a))

	g.GET("/dismantles", requirePerm(a.User, "menu:dismantle"), orderListDismantlesHandler(a))
	g.POST("/dismantles", requirePerm(a.User, "menu:dismantle"), orderCreateDismantleHandler(a))

	g.GET("/activation-callbacks", requirePerm(a.User, "menu:callback"), orderListActivationCallbacksHandler(a))
	g.POST("/activation-callbacks/:callbackId/retry", requirePerm(a.User, "menu:callback"), orderRetryActivationCallbackHandler(a))
}