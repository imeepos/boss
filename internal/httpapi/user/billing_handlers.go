package userapi

// 用户端门户 Billing 域:handler 实现(billing.go 仅留路由表 + 数据视图)。

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/promotion"
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
// 可选 couponId:缴费同事务核销,实收=账单金额-抵扣。
func portalCreatePayment(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req portalPayReq
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

// portalPayReq 缴费请求体。
type portalPayReq struct {
	BillNo    string  `json:"billNo" binding:"required"`
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	PayMethod string  `json:"payMethod" binding:"required,oneof=wechat alipay card cash"`
	CouponID  string  `json:"couponId"`
	// 预缴场景:实购 N 个月(productId 可空=全店规则),命中赠送阶梯时落痕并返回 giftMonths。
	BuyMonths int   `json:"buyMonths"`
	ProductID int64 `json:"productId"`
}

// respondPay 落单笔缴费(PAID 冲突检测 + RecordPayment;带券走同事务核销;
// buyMonths>0 时按赠送阶梯规则落痕)。
func respondPay(c *gin.Context, a *app.Application, b billing.Bill, req portalPayReq) {
	if b.Status == "PAID" {
		respond(c, apitypes.CodeConflict, nil)
		return
	}
	payNo, err := a.Portal.NextNo(c.Request.Context(), "PAY")
	if err != nil {
		respondErr(c, err)
		return
	}
	receipt, err := a.Billing.RecordPaymentWithCoupon(c.Request.Context(), billing.Payment{
		PayNo: payNo, BillID: b.BillID, CustomerID: b.CustomerID, Amount: req.Amount,
		Method: req.PayMethod, Status: "SUCCESS", CouponID: req.CouponID,
	})
	if err != nil {
		respondErr(c, err)
		return
	}
	giftMonths := recordDurationGift(c, a, b.CustomerID, req, receipt.PaymentID)
	earnPointsForPayment(c.Request.Context(), a, b.CustomerID, receipt.PaymentID, req.Amount)
	a.ResumeAfterPayment(c.Request.Context(), b.CustomerID)
	respond(c, apitypes.CodeOK, gin.H{
		"payNo": payNo, "amount": receipt.Amount, "billPeriod": b.Period,
		"payMethod": req.PayMethod, "status": "SUCCESS",
		"deductedCents": receipt.DeductedCents, "paymentId": receipt.PaymentID,
		"giftMonths": giftMonths,
	})
}

// recordDurationGift 缴费成功后赠送时长落痕:命中阶梯返回赠送月数,未命中/未配置返回 0。
func recordDurationGift(c *gin.Context, a *app.Application, cid int64, req portalPayReq, paymentID int64) int {
	if req.BuyMonths <= 0 || a.Promotion == nil {
		return 0
	}
	rule, err := a.Promotion.MatchGiftRule(c.Request.Context(), req.ProductID, req.BuyMonths)
	if err != nil || rule == nil {
		return 0
	}
	if err := a.Promotion.RecordGift(c.Request.Context(), promotion.GiftRecord{
		RuleID: rule.RuleID, CustomerID: cid, ProductID: req.ProductID,
		BuyMonths: req.BuyMonths, GiftMonths: rule.GiftMonths, PaymentID: paymentID,
	}); err != nil {
		return 0
	}
	return rule.GiftMonths
}

// portalListPayments GET /payments:我的缴费记录(默认仅 SUCCESS,
// ?include=failed,refunded 时把 FAILED/REFUNDED 一并返回)。
//
// 业务口径(terms.md §4 payment.status 枚举):SUCCESS 是对用户有意义的"缴费完成"事件;
// FAILED 仅留痕待人工排查,REFUNDED 由财务侧冲账,不向终端用户展示默认列表。
// 状态字段无论是否过滤都回传,便于前端识别"被过滤掉"的项。
func portalListPayments(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		pays, err := a.Billing.ListPaymentsByCustomer(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		includeFailed := strings.Contains(c.Query("include"), "failed")
		includeRefunded := strings.Contains(c.Query("include"), "refunded")
		periodByBill := portalBillPeriods(a, c, cid)
		items := make([]gin.H, 0, len(pays))
		for _, p := range pays {
			if p.Status != billing.PaymentStatusSuccess && !portalPaymentIncluded(p, includeFailed, includeRefunded) {
				continue
			}
			items = append(items, gin.H{
				"payNo": p.PayNo, "amount": p.Amount,
				"period":    portalPaymentPeriod(p, periodByBill),
				"payMethod": p.Method,
				"paidAt":    time.Now().Format(time.RFC3339),
				"status":    p.Status,
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// portalPaymentIncluded 失败/退款类流水在 include 标志下的可见性判定。
// includeFailed=false 时 FAILED 一律不展示(免用户误解为已缴费);
// includeRefunded=false 时 REFUNDED 不展示(财务冲账流程不向终端用户解释)。
func portalPaymentIncluded(p billing.Payment, includeFailed, includeRefunded bool) bool {
	switch p.Status {
	case billing.PaymentStatusFailed:
		return includeFailed
	case billing.PaymentStatusRefunded:
		return includeRefunded
	}
	return true
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

// portalBalanceGet / portalTopup / topupRecordPayment / portalTopupReq 已迁至 billing_topup.go
// (超 300 行拆分,handler 接口不变)。
