package userapi

// 用户端门户 Billing 域:handler 实现(billing.go 仅留路由表 + 数据视图)。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalListBills GET /bills?status=:我的账单列表(currentDue/currentPeriod/items)。
func portalListBills(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		status := c.DefaultQuery("status", "recent")
		bills, err := a.Billing.ListBills(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		currentDue, currentPeriod, items := portalBillsAggregate(bills, status, a, c, cid)
		respond(c, apitypes.CodeOK, gin.H{
			"currentDue": currentDue, "currentPeriod": currentPeriod, "items": items,
		})
	}
}

// portalBillsAggregate 按 status 过滤 + 累计 currentDue/currentPeriod(items)。
func portalBillsAggregate(bills []billing.Bill, status string, a *app.Application, c *gin.Context, cid int64) (float64, string, []gin.H) {
	currentDue, currentPeriod := 0.0, ""
	items := make([]gin.H, 0, len(bills))
	productName := planProductName(a, c, cid)
	for _, b := range bills {
		if !portalBillStatusMatch(b, status) {
			continue
		}
		items = append(items, portalBill(b, productName))
		if b.Status != "PAID" {
			currentDue += b.Amount
		}
		if b.Period > currentPeriod {
			currentPeriod = b.Period
		}
	}
	return currentDue, currentPeriod, items
}

// portalBillStatusMatch 账单是否匹配 status 过滤(recent/unpaid 排除已缴;paid 只看已缴;其余两段都看)。
func portalBillStatusMatch(b billing.Bill, status string) bool {
	if b.Status == "PAID" {
		return status != "unpaid" && status != "recent"
	}
	return status != "paid"
}

// portalCreatePayment POST /payments:发起缴费(bill 归属校验 + RecordPayment 落库置 PAID)。
func portalCreatePayment(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			BillNo    string  `json:"billNo" binding:"required"`
			Amount    float64 `json:"amount" binding:"required,gt=0"`
			PayMethod string  `json:"payMethod" binding:"required,oneof=wechat alipay card cash"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		bills, err := a.Billing.ListBills(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		for _, b := range bills {
			if b.BillNo != req.BillNo {
				continue
			}
			respondPay(c, a, b, req)
			return
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}

// respondPay 落单笔缴费(PAID 冲突检测 + RecordPayment)。
func respondPay(c *gin.Context, a *app.Application, b billing.Bill, req struct {
	BillNo    string  `json:"billNo" binding:"required"`
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	PayMethod string  `json:"payMethod" binding:"required,oneof=wechat alipay card cash"`
}) {
	if b.Status == "PAID" {
		respond(c, apitypes.CodeConflict, nil)
		return
	}
	payNo, err := a.Portal.NextNo(c.Request.Context(), "PAY")
	if err != nil {
		respondErr(c, err)
		return
	}
	if _, err := a.Billing.RecordPayment(c.Request.Context(), billing.Payment{
		PayNo: payNo, BillID: b.BillID, Amount: req.Amount,
		Method: req.PayMethod, Status: "SUCCESS",
	}); err != nil {
		respondErr(c, err)
		return
	}
	respond(c, apitypes.CodeOK, gin.H{
		"payNo": payNo, "amount": req.Amount, "billPeriod": b.Period,
		"payMethod": req.PayMethod, "status": "SUCCESS",
	})
}

// portalListPayments GET /payments:我的缴费记录(缴费+充值,按客户聚合)。
func portalListPayments(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		pays, err := a.Billing.ListPaymentsByCustomer(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		periodByBill := portalBillPeriods(a, c, cid)
		items := make([]gin.H, 0, len(pays))
		for _, p := range pays {
			items = append(items, gin.H{
				"payNo": p.PayNo, "amount": p.Amount,
				"period":     portalPaymentPeriod(p, periodByBill),
				"payMethod":  p.Method,
				"paidAt":     time.Now().Format(time.RFC3339),
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// portalBillDetail GET /bills/:billNo:我的账单明细(按客户过滤)。
func portalBillDetail(a *app.Application) gin.HandlerFunc {
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

// portalReceipt GET /payments/:payNo/receipt:在我的缴费流水中找凭证(含充值)。
func portalReceipt(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		pays, err := a.Billing.ListPaymentsByCustomer(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		periodByBill := portalBillPeriods(a, c, cid)
		for _, p := range pays {
			if p.PayNo == c.Param("payNo") {
				respond(c, apitypes.CodeOK, gin.H{
					"receiptNo": "OR-" + p.PayNo, "amount": p.Amount,
					"period":    portalPaymentPeriod(p, periodByBill),
					"payMethod": p.Method, "status": p.Status,
					"paidAt": time.Now().Format(time.RFC3339), "payNo": p.PayNo,
				})
				return
			}
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}

// portalBalanceGet GET /topups:余额查询。
func portalBalanceGet(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		bal, err := a.Portal.Balance(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"balance": bal, "denominations": []int{50, 100, 200}})
	}
}

// portalTopup POST /topups:充值入余额(支付通道接入前仅记账;余额/单号已落库)。
func portalTopup(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req portalTopupReq
		if !httpx.BindBody(c, &req) {
			return
		}
		if err := a.Portal.AdjustBalance(c.Request.Context(), cid, req.Amount); err != nil {
			respondErr(c, err)
			return
		}
		payNo, ok := topupRecordPayment(c, a, cid, req)
		if !ok {
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"payNo": payNo, "amount": req.Amount, "payMethod": req.PayMethod, "status": "SUCCESS",
		})
	}
}
// portalTopupReq 充值请求体。
type portalTopupReq struct {
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	PayMethod string  `json:"payMethod" binding:"required"`
}

// topupRecordPayment 充值派单号 + 落缴费流水(bill_id NULL + customer_id 归属,
// 否则 /payments 与凭证端点查不到);失败已回写响应。
func topupRecordPayment(c *gin.Context, a *app.Application, cid int64, req portalTopupReq) (string, bool) {
	payNo, err := a.Portal.NextNo(c.Request.Context(), "PAY")
	if err != nil {
		respondErr(c, err)
		return "", false
	}
	if _, err := a.Billing.CreatePayment(c.Request.Context(), billing.Payment{
		PayNo: payNo, CustomerID: cid, Amount: req.Amount,
		Method: req.PayMethod, Status: "SUCCESS",
	}); err != nil {
		respondErr(c, err)
		return "", false
	}
	return payNo, true
}
