package adminapi

// 订单域子表路由 handler 实现(承接 registerOrderSubRoutes)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// orderListComplaintsHandler GET /complaints:报障与投诉列表(CS 客服域)。
func orderListComplaintsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkOrder.ListComplaints(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// orderCloseComplaintHandler POST /complaints/{ticketNo}/close:办结报障。
func orderCloseComplaintHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := a.WorkOrder.CloseComplaint(c.Request.Context(), c.Param("ticketNo")); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "org.update", "complaint", c.Param("ticketNo"), map[string]any{"op": "close"})
		resolveTodo(c.Request.Context(), a, refComplaint, c.Param("ticketNo"))
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// orderListDismantlesHandler GET /dismantles:拆机单列表。
func orderListDismantlesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.OrderLedger.ListDismantles(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// orderCreateDismantleHandler POST /dismantles:新建拆机单。
func orderCreateDismantleHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// orderListActivationCallbacksHandler GET /activation-callbacks:激活回调列表(订单第 11 环节)。
func orderListActivationCallbacksHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.OrderLedger.ListActivationCallbacks(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// orderRetryActivationCallbackHandler POST /activation-callbacks/{callbackId}/retry:激活回调重试。
func orderRetryActivationCallbackHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}
