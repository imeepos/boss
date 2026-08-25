package workerapi

// W 师傅端门户扫码绑定闭环(worker/scan.yaml):环节9/10、取证、签收、现场收款。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerWorkerPortalScanRoutes 扫码绑定域路由(wauth 组)。
func registerWorkerPortalScanRoutes(g *gin.RouterGroup, a *app.Application) {
	g.POST("/tickets/:ticketNo/scan-bind", workerScanBindHandler(a))
	g.POST("/tickets/:ticketNo/scan-abnormal", workerAuditOK(a, "scan-abnormal"))
	g.GET("/tickets/:ticketNo/photos", workerPhotoListHandler)
	g.POST("/tickets/:ticketNo/photos", workerPhotoUploadHandler(a))
	g.GET("/tickets/:ticketNo/report", workerReportGetHandler(a))
	g.POST("/tickets/:ticketNo/report", workerReportSubmitHandler(a))
	g.GET("/tickets/:ticketNo/activation", workerActivationGetHandler(a))
	g.POST("/tickets/:ticketNo/activate", workerActivateHandler(a))
	g.POST("/tickets/:ticketNo/sign", workerAuditOK(a, "sign"))
	g.GET("/tickets/:ticketNo/charge", workerChargeGetHandler)
	g.POST("/tickets/:ticketNo/charge", workerChargePostHandler(a))
}

// workerPhotoListHandler 取证列表(预取空集,客户端按文件上传后端刷新)。
func workerPhotoListHandler(c *gin.Context) {
	respond(c, apitypes.CodeOK, gin.H{"items": []gin.H{}})
}

// portalQuadH 按地址取四码对照视图:码值经绑定链解析(资产码/端口码真实,
// 用户地址码取 user_addresses.addr_code;customerCode 取 customers.customer_code,
// adopted 2026-08-21 后由 migration 000085 + BEFORE INSERT 触发器自动派生)。
func portalQuadH(a *app.Application, c *gin.Context, addressID int64) gin.H {
	q, err := a.QuadLink.GetByAddress(c.Request.Context(), addressID)
	if err != nil || q == nil {
		return gin.H{"status": "UNLINKED", "matched": false,
			"assetCode": "", "customerCode": "", "portCode": "", "addrCode": ""}
	}
	return gin.H{
		"status": q.Status, "matched": q.Status == "LINKED",
		"assetCode":    quadAssetCode(a, c, q.AssetID),
		"portCode":     quadPortCode(a, c, q.PortID),
		"addrCode":     quadAddrCode(a, c, addressID),
		"customerCode": quadCustomerCode(a, c, q.CustomerID),
	}
}

// quadCustomerCode 用户码(customers.customer_code,adopted 2026-08-21)。
func quadCustomerCode(a *app.Application, c *gin.Context, customerID int64) string {
	cust, err := a.Customer.Get(c.Request.Context(), customerID)
	if err != nil || cust == nil {
		return ""
	}
	return cust.CustomerCode
}

// quadAssetCode 资产码(assets.asset_code)。
func quadAssetCode(a *app.Application, c *gin.Context, assetID int64) string {
	ast, err := a.Asset.GetAsset(c.Request.Context(), assetID)
	if err != nil || ast == nil {
		return ""
	}
	return ast.AssetCode
}

// quadPortCode 端口码(ports.port_code;单查 GetPort,非遍历)。
func quadPortCode(a *app.Application, c *gin.Context, portID int64) string {
	p, err := a.Resource.GetPort(c.Request.Context(), portID)
	if err != nil || p == nil {
		return ""
	}
	return p.PortCode
}

// quadAddrCode 地址码(addresses.name,权威表;如"马尼拉市")。
func quadAddrCode(a *app.Application, c *gin.Context, addressID int64) string {
	ad, err := a.Geo.GetAddress(c.Request.Context(), addressID)
	if err != nil || ad == nil {
		return ""
	}
	return ad.Name
}

// workerReportGetHandler 上报预取:四码对照 + 检测项占位。
func workerReportGetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, ord, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"ticketNo": tk.TicketNo, "quad": portalQuadH(a, c, ord.AddressID),
			"checks":       gin.H{"powerOn": true, "opticalPowerDbm": 0, "provisionDone": true, "loidAuthPassed": true},
			"provisionLog": gin.H{"template": "", "preResult": "", "onsiteResult": ""},
		})
	}
}

// workerReportSubmitHandler 提交装维结果上报(环节10 激活)。
func workerReportSubmitHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, _, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !workerOwnedTicket(c, tk) {
			return
		}
		if err := a.Order.ActivateUser(c.Request.Context(), tk.OrderID); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// activationState 激活状态视图:订单环节 >10 视为 SUCCESS,=10 为 PENDING。
func activationState(ticketNo string, stage int8) gin.H {
	status := "PENDING"
	if stage > 10 {
		status = "SUCCESS"
	}
	return gin.H{"ticketNo": ticketNo, "loid": "", "status": status, "statusLabel": "", "lastTry": ""}
}

// workerActivationGetHandler 激活状态查询。
func workerActivationGetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, ord, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, activationState(tk.TicketNo, ord.Stage))
	}
}

// workerActivateHandler 重新激活(环节10):失败可反复重试。
func workerActivateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, _, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !workerOwnedTicket(c, tk) {
			return
		}
		if err := a.Order.ActivateUser(c.Request.Context(), tk.OrderID); err != nil {
			respondErr(c, err)
			return
		}
		ord, _, err := a.Order.Track(c.Request.Context(), tk.OrderID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, activationState(tk.TicketNo, ord.Stage))
	}
}

// workerChargeGetHandler 现场收款预取:订单价格快照未暴露,金额待 DB 商议。
func workerChargeGetHandler(c *gin.Context) {
	respond(c, apitypes.CodeOK, gin.H{
		"ticketNo": c.Param("ticketNo"), "amountDue": 0, "amountDesc": "",
		"payMethods": []string{"QR", "CASH", "POS"},
	})
}

// workerChargeReq 现场收款确认。
type workerChargeReq struct {
	Amount    float64 `json:"amount" binding:"required"`
	PayMethod string  `json:"payMethod" binding:"required"`
}

// workerChargePostHandler 确认收款:账务联动缺口见报告,先审计留痕。
func workerChargePostHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, _, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !workerOwnedTicket(c, tk) {
			return
		}
		var req workerChargeReq
		if !httpx.BindAndValidate(c, &req, func() error {
			if req.Amount <= 0 {
				return &httpx.ValidationError{Field: "amount", Message: "must be greater than 0"}
			}
			return nil
		}) {
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "worker_charge", c.Param("ticketNo"), map[string]any{
			"amount": req.Amount, "payMethod": req.PayMethod,
		})
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "payNo": "", "receiptUrl": ""})
	}
}

// ticketOrder 按工单号取工单 + 订单(门户内高频组合)。
func ticketOrder(c *gin.Context, a *app.Application) (*order.DispatchTicket, *order.Order, error) {
	tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), c.Param("ticketNo"))
	if err != nil {
		return nil, nil, err
	}
	ord, _, err := a.Order.Track(c.Request.Context(), tk.OrderID)
	if err != nil {
		return nil, nil, err
	}
	return tk, ord, nil
}
