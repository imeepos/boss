package adminapi

// 开网订单流程推进端点(CLI 模拟 / 联调用):环节 2/3/4 人工推进 + 收费后自动段 5-8 + 激活后自动段 10-12。
// 验收口径(terms.md §1 12 环节):人工只做资源核查(2)、端口预占(3)、合同收费(4)与上门激活(10);
// 标签预绑定/建档/预下发/派单(5-8)与回调/GIS(11-12)由 Automation 自动推进。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerOrderWorkflowRoutes 注册订单流程推进路由(承接 order.yaml workflow 段)。
func registerOrderWorkflowRoutes(g *gin.RouterGroup, a *app.Application) {
	wf := g.Group("/orders", requirePerm(a.User, "menu:order"))

	// 核查预览:只读查询地址下设备与端口分布,不推进订单。
	wf.GET("/:orderNo/check-preview", func(c *gin.Context) {
		o, err := a.Order.GetByNo(c.Request.Context(), c.Param("orderNo"))
		if err != nil { respondErr(c, err); return }
		detail, err := a.Resource.CheckDetail(c.Request.Context(), o.AddressID)
		if err != nil { respondErr(c, err); return }
		respond(c, apitypes.CodeOK, gin.H{"orderNo": o.OrderNo, "addressId": o.AddressID, "stage": o.Stage, "status": o.Status, "detail": detail})
	})

	// 环节2 资源核查:调资源域核查目标地址,推进 stage=2。
	wf.POST("/:orderNo/check-resource", func(c *gin.Context) {
		o, err := a.Order.GetByNo(c.Request.Context(), c.Param("orderNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		available, idlePorts, err := a.Resource.Check(c.Request.Context(), o.AddressID)
		if err != nil { respondErr(c, err); return }
		if err := a.Order.CheckResource(c.Request.Context(), o.ID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "order.check_resource", "order", o.OrderNo, gin.H{"available": available, "idlePorts": idlePorts})
		respond(c, apitypes.CodeOK, gin.H{"orderNo": o.OrderNo, "stage": 2, "available": available, "idlePorts": idlePorts})
	})

	// 环节3 端口预占:资源域占空闲端口(写 order_id),再推进状态机 PENDING→RESERVED。
	wf.POST("/:orderNo/reserve", func(c *gin.Context) {
		o, err := a.Order.GetByNo(c.Request.Context(), c.Param("orderNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		portID, err := a.Resource.ReserveFirstAvailable(c.Request.Context(), o.AddressID, o.ID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Order.Reserve(c.Request.Context(), o.ID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "order.reserve", "order", o.OrderNo, gin.H{"portId": portID})
		respond(c, apitypes.CodeOK, gin.H{"orderNo": o.OrderNo, "stage": 3, "portId": portID})
	})

	// 环节4 合同收费 + 自动段 5-8(标签预绑定/建档/预下发/派单):未收费不派单由顺序守卫保证。
	wf.POST("/:orderNo/charge", func(c *gin.Context) {
		o, err := a.Order.GetByNo(c.Request.Context(), c.Param("orderNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Order.ChargeContract(c.Request.Context(), o.ID); err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Automation.AutoPreScan(c.Request.Context(), o.ID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "order.charge", "order", o.OrderNo, nil)
		respond(c, apitypes.CodeOK, gin.H{"orderNo": o.OrderNo, "stage": 8})
	})

	// 订单取消:任一未完成状态可取消(status→CANCELLED),与门户 portalOrderCancel 同语义。
	wf.POST("/:orderNo/cancel", func(c *gin.Context) {
		o, err := a.Order.GetByNo(c.Request.Context(), c.Param("orderNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Order.Cancel(c.Request.Context(), o.ID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "order.cancel", "order", o.OrderNo, nil)
		respond(c, apitypes.CodeOK, gin.H{"orderNo": o.OrderNo, "status": "CANCELLED"})
	})

	// 环节10 激活(上门扫码后,师傅上报装维结果):自动段 10-12(激活/回调/更新GIS)→ 订单 DONE。
	// 与扫码路由同门禁:任何已认证主体可调(worker 主体/账号),身份经 Subject 或 claims 识别。
	tic := g.Group("/tickets")
	tic.POST("/:ticketNo/activate", func(c *gin.Context) {
		tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), c.Param("ticketNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Automation.AutoPostScan(c.Request.Context(), tk.OrderID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "order.activate", "dispatch_ticket", tk.TicketNo, nil)
		respond(c, apitypes.CodeOK, gin.H{"ticketNo": tk.TicketNo, "ok": true})
	})
}
