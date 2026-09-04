package workerapi

// W 师傅端门户扫码绑定闭环(worker/scan.yaml):环节9/10、取证、签收、现场收款。
// 激活检测字段只反映已确认的订单状态，未接入网元/AAA 时不得伪造成功。

import (
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/worker"
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
	g.POST("/tickets/:ticketNo/sign", workerSignHandler(a))
	g.GET("/tickets/:ticketNo/charge", workerChargeGetHandler(a))
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

// workerReportGetHandler 上报预取:四码对照 + 当前真实可确认状态。
func workerReportGetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, ord, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"ticketNo": tk.TicketNo, "quad": portalQuadH(a, c, ord.AddressID),
			"checks":       gin.H{"powerOn": false, "opticalPowerDbm": 0, "provisionDone": false, "loidAuthPassed": false},
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
		if err := activateWorkerOrder(c, a, tk.OrderID); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// activateWorkerOrder 激活统一入口：扫码后 AutoPostScan 自动段 10-12(与 admin 一致)。
func activateWorkerOrder(c *gin.Context, a *app.Application, orderID int64) error {
	ord, _, err := a.Order.Track(c.Request.Context(), orderID)
	if err != nil {
		return err
	}
	if ord.Stage < 9 {
		// 未扫码绑定(环节9)不可激活:httpx 映射 40910(此前裸 error → 50000)。
		return worker.ErrScanBindRequired
	}
	if a.Automation != nil {
		err := a.Automation.AutoPostScan(c.Request.Context(), orderID)
		if err == nil {
			return nil
		}
		if activationLegFailed(err) {
			recordFailedAttempt(c, a, orderID) // 失败尝试落痕(任务A-d)
		}
		return err
	}
	if ord.Stage == 9 {
		if err := a.Order.ActivateUser(c.Request.Context(), orderID); err != nil {
			recordFailedAttempt(c, a, orderID)
			return err
		}
	}
	return nil
}

// activationLegFailed 自动段失败是否发生在激活腿(10 激活/11 回执)——
// updateMap(12)失败不代表激活失败,不计 FAILED 回执。
func activationLegFailed(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "automation: activateUser:") ||
		strings.Contains(msg, "automation: notifyActivation:")
}

// recordFailedAttempt 激活失败尝试落痕(任务A-d):FAILED 回执幂等 upsert。
// 尽力而为,写失败只留 ALERT 不阻断原错误返回。
func recordFailedAttempt(c *gin.Context, a *app.Application, orderID int64) {
	if a.OrderLedger == nil {
		return
	}
	if _, err := a.OrderLedger.AppendActivationCallback(c.Request.Context(),
		order.ActivationCallback{OrderID: orderID, Result: "FAILED"}); err != nil {
		log.Printf("[worker-activation] ATTEMPT RECORD FAILED order=%d err=%v", orderID, err)
	}
}

// latestActivationTry 最近一次激活尝试时刻(任务A-d lastTry);无记录返回空串。
func latestActivationTry(c *gin.Context, a *app.Application, orderID int64) string {
	if a.OrderLedger == nil {
		return ""
	}
	cb, err := a.OrderLedger.LatestActivationCallback(c.Request.Context(), orderID)
	if err != nil || cb == nil || cb.TriedAt == nil {
		return ""
	}
	return cb.TriedAt.Format(time.RFC3339)
}

// activationState 激活状态视图：只有环节11完成才代表订单侧回调成功。
// lastTry 为最近一次激活尝试时刻(RFC3339;空=无尝试记录)。
func activationState(ticketNo string, stage int8, loid string, lastTry string) gin.H {
	status := "PENDING"
	if stage >= 11 {
		status = "SUCCESS"
	}
	return gin.H{"ticketNo": ticketNo, "loid": loid, "status": status, "statusLabel": "", "lastTry": lastTry}
}

// workerOrderLoid 按客户查 LOID;未建档或不可用时返回空串。
func workerOrderLoid(c *gin.Context, a *app.Application, customerID int64) string {
	if customerID <= 0 || a.Aaa == nil {
		return ""
	}
	lo, err := a.Aaa.GetLoAccountByCustomer(c.Request.Context(), customerID)
	if err != nil || lo == nil {
		return ""
	}
	return lo.Loid
}

// workerActivationGetHandler 激活状态查询。
func workerActivationGetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, ord, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		loid := workerOrderLoid(c, a, ord.CustomerID)
		respond(c, apitypes.CodeOK, activationState(tk.TicketNo, ord.Stage, loid, latestActivationTry(c, a, tk.OrderID)))
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
		if err := activateWorkerOrder(c, a, tk.OrderID); err != nil {
			respondErr(c, err)
			return
		}
		ord, _, err := a.Order.Track(c.Request.Context(), tk.OrderID)
		if err != nil {
			respondErr(c, err)
			return
		}
		loid := workerOrderLoid(c, a, ord.CustomerID)
		respond(c, apitypes.CodeOK, activationState(tk.TicketNo, ord.Stage, loid, latestActivationTry(c, a, tk.OrderID)))
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
