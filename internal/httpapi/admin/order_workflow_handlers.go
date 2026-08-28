package adminapi

// 订单流程推进路由具名 handler(承接 registerOrderWorkflowRoutes 扁平路由表)。
// 人工段 2/3/4/10 + 取消 + 派单激活;自动段 5-8 与 11-12 由 Automation 推进。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// orderCheckPreviewHandler GET /orders/{orderNo}/check-preview:只读查询地址下设备/端口分布,不推进订单。
func orderCheckPreviewHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		o, err := a.Order.GetByNo(c.Request.Context(), c.Param("orderNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		if !requireOrderInScope(c, a, o) {
			return
		}
		detail, err := a.Resource.CheckDetail(c.Request.Context(), o.AddressID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"orderNo": o.OrderNo, "addressId": o.AddressID, "stage": o.Stage, "status": o.Status, "detail": detail})
	}
}

// orderCheckResourceHandler POST /orders/{orderNo}/check-resource:环节2 资源核查,推进 stage=2。
func orderCheckResourceHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		o, err := a.Order.GetByNo(c.Request.Context(), c.Param("orderNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		if !requireOrderInScope(c, a, o) {
			return
		}
		available, idlePorts, err := a.Resource.Check(c.Request.Context(), o.AddressID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Order.CheckResource(c.Request.Context(), o.ID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "order.check_resource", "order", o.OrderNo, gin.H{"available": available, "idlePorts": idlePorts})
		respond(c, apitypes.CodeOK, gin.H{"orderNo": o.OrderNo, "stage": 2, "available": available, "idlePorts": idlePorts})
	}
}

// orderReserveHandler POST /orders/{orderNo}/reserve:环节3 端口预占;状态机拒绝时回滚端口防泄漏。
func orderReserveHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		o, err := a.Order.GetByNo(c.Request.Context(), c.Param("orderNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		if !requireOrderInScope(c, a, o) {
			return
		}
		portID, err := a.Resource.ReserveFirstAvailable(c.Request.Context(), o.AddressID, o.ID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Order.Reserve(c.Request.Context(), o.ID); err != nil {
			// 补偿:状态机拒绝(如重复预占)时回滚刚占的端口,避免 RESERVED 端口泄漏。
			if rerr := a.Resource.ReleasePortByOrder(c.Request.Context(), o.ID); rerr != nil {
				httpx.RecordAudit(a, c, "order.reserve_compensate_failed", "order", o.OrderNo,
					gin.H{"portId": portID, "error": rerr.Error()})
			}
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "order.reserve", "order", o.OrderNo, gin.H{"portId": portID})
		respond(c, apitypes.CodeOK, gin.H{"orderNo": o.OrderNo, "stage": 3, "portId": portID})
	}
}

// orderChargeHandler POST /orders/{orderNo}/charge:环节4 合同收费 + 自动段 5-8;幂等(stage>=4 跳过 Charge)。
func orderChargeHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		o, err := a.Order.GetByNo(c.Request.Context(), c.Param("orderNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		if !requireOrderInScope(c, a, o) {
			return
		}
		if o.Stage < 4 {
			if err := a.Order.ChargeContract(c.Request.Context(), o.ID); err != nil {
				respondErr(c, err)
				return
			}
		}
		if err := a.Automation.AutoPreScan(c.Request.Context(), o.ID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "order.charge", "order", o.OrderNo, nil)
		respond(c, apitypes.CodeOK, gin.H{"orderNo": o.OrderNo, "stage": 8})
	}
}

// orderCancelHandler POST /orders/{orderNo}/cancel:任一未完成状态可取消,与门户 portalOrderCancel 同语义。
func orderCancelHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		o, err := a.Order.GetByNo(c.Request.Context(), c.Param("orderNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		if !requireOrderInScope(c, a, o) {
			return
		}
		if err := a.Order.Cancel(c.Request.Context(), o.ID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "order.cancel", "order", o.OrderNo, nil)
		respond(c, apitypes.CodeOK, gin.H{"orderNo": o.OrderNo, "status": "CANCELLED"})
	}
}

// orderActivateHandler POST /tickets/{ticketNo}/activate:环节10 师傅上报装维结果,自动段 10-12 推进订单 DONE。
func orderActivateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), c.Param("ticketNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		if !requireTicketInScope(c, a, tk) {
			return
		}
		if err := a.Automation.AutoPostScan(c.Request.Context(), tk.OrderID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "order.activate", "dispatch_ticket", tk.TicketNo, nil)
		respond(c, apitypes.CodeOK, gin.H{"ticketNo": tk.TicketNo, "ok": true})
	}
}
