package app

// 用户端门户 Billing 域:账单明细/缴费凭证/余额充值/申请开票 + 门户偏好存取。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/pkg/apitypes"
)

// getPrefs/savePrefs 门户偏好(通知订阅/语言)进程内存取;默认值内置。
func (s *portalStore) getPrefs(customerID int64) *portalPrefs {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.prefs[customerID]
	if p == nil {
		p = &portalPrefs{Notify: gin.H{
			"business":  gin.H{"bill": true, "order": true},
			"marketing": gin.H{"promo": false},
			"channels":  gin.H{"sms": true, "app": true, "email": false},
		}, Language: "zh"}
		s.prefs[customerID] = p
	}
	return p
}

func (s *portalStore) savePrefs(customerID int64, notify gin.H, language string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.prefs[customerID]
	if p == nil {
		p = &portalPrefs{}
		s.prefs[customerID] = p
	}
	if notify != nil {
		p.Notify = notify
	}
	if language != "" {
		p.Language = language
	}
}

// registerPortalBillingRoutes 账单明细/凭证/充值/开票。
func registerPortalBillingRoutes(g *gin.RouterGroup, a *Application) {
	g.GET("/bills/:billNo", portalBillDetail(a))
	g.GET("/payments/:payNo/receipt", portalReceipt(a))
	g.GET("/topups", portalBalanceGet)
	g.POST("/topups", portalTopup)
	g.POST("/invoices", portalApplyInvoice(a))
}

// portalBillDetail GET /bills/:billNo:我的账单明细(按客户过滤)。
func portalBillDetail(a *Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		bills, err := a.Billing.ListBills(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		for _, b := range bills {
			if b.BillNo == c.Param("billNo") {
				respond(c, apitypes.CodeOK, gin.H{
					"bill":     gin.H{"billNo": b.BillNo, "period": b.Period, "amount": b.Amount, "status": b.Status},
					"items":    []gin.H{{"name": "宽带月费", "range": b.Period, "amount": b.Amount}},
					"totalDue": map[bool]float64{true: 0, false: b.Amount}[b.Status == "PAID"],
				})
				return
			}
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}

// portalReceipt GET /payments/:payNo/receipt:在我的缴费流水中找凭证。
func portalReceipt(a *Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		bills, _ := a.Billing.ListBills(c.Request.Context(), cid)
		for _, b := range bills {
			pays, _ := a.Billing.ListPayments(c.Request.Context(), b.BillID)
			for _, p := range pays {
				if p.PayNo == c.Param("payNo") {
					respond(c, apitypes.CodeOK, gin.H{
						"receiptNo": "OR-" + p.PayNo, "amount": p.Amount,
						"period": b.Period, "payMethod": p.Method, "status": p.Status,
						"paidAt": time.Now().Format(time.RFC3339), "payNo": p.PayNo,
					})
					return
				}
			}
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}

func portalBalanceGet(c *gin.Context) {
	cid, _ := requireCustomer(c)
	respond(c, apitypes.CodeOK, gin.H{"balance": portal.balance(cid), "denominations": []int{50, 100, 200}})
}

// portalTopup POST /topups:充值入余额(支付通道接入前仅记账;见报告)。
func portalTopup(c *gin.Context) {
	cid, _ := requireCustomer(c)
	var req struct {
		Amount    float64 `json:"amount" binding:"required,gt=0"`
		PayMethod string  `json:"payMethod" binding:"required"`
	}
	if !bindBody(c, &req) {
		return
	}
	portal.adjustBalance(cid, req.Amount)
	respond(c, apitypes.CodeOK, gin.H{
		"payNo": portal.payNo(), "amount": req.Amount, "payMethod": req.PayMethod, "status": "SUCCESS",
	})
}

// portalApplyInvoice POST /invoices:申请开票(校验账单归属;Tax 域客户侧开票流程见报告)。
func portalApplyInvoice(a *Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			BillNo string `json:"billNo" binding:"required"`
		}
		if !bindBody(c, &req) {
			return
		}
		bills, _ := a.Billing.ListBills(c.Request.Context(), cid)
		for _, b := range bills {
			if b.BillNo == req.BillNo {
				respond(c, apitypes.CodeOK, gin.H{"ok": true})
				return
			}
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}
