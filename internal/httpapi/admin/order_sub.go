package adminapi

// 订单域子表路由:报障投诉/拆机/激活回调(order.yaml 已声明、原未实现的 5 个端点)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerOrderSubRoutes 注册订单子表路由(承接 api/openapi/admin/order.yaml)。
func registerOrderSubRoutes(g *gin.RouterGroup, a *app.Application) {
	// 报障与投诉(CS 客服域):列表 + 办结。
	g.GET("/complaints", requirePerm(a.User, "menu:complaint"), func(c *gin.Context) {
		list, err := a.WorkOrder.ListComplaints(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	g.POST("/complaints/:ticketNo/close", requirePerm(a.User, "menu:complaint"), func(c *gin.Context) {
		if err := a.WorkOrder.CloseComplaint(c.Request.Context(), c.Param("ticketNo")); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "org.update", "complaint", c.Param("ticketNo"), map[string]any{"op": "close"})
		resolveTodo(c.Request.Context(), a, refComplaint, c.Param("ticketNo"))
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	// 拆机单:列表 + 新建。
	g.GET("/dismantles", requirePerm(a.User, "menu:dismantle"), func(c *gin.Context) {
		list, err := a.OrderLedger.ListDismantles(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	g.POST("/dismantles", requirePerm(a.User, "menu:dismantle"), func(c *gin.Context) {
		var d order.Dismantle
		if !httpx.BindAndValidate(c, &d) {
			return
		}
		id, err := a.OrderLedger.CreateDismantle(c.Request.Context(), d)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	// 激活回调(订单第 11 环节):列表 + 重试。
	g.GET("/activation-callbacks", requirePerm(a.User, "menu:callback"), func(c *gin.Context) {
		list, err := a.OrderLedger.ListActivationCallbacks(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	g.POST("/activation-callbacks/:callbackId/retry", requirePerm(a.User, "menu:callback"), func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "callbackId")
		if !ok {
			return
		}
		if err := a.OrderLedger.RetryActivationCallback(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "org.update", "activation_callback", c.Param("callbackId"), map[string]any{"op": "retry"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
}
