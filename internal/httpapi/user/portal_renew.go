package userapi

// 套餐续费:预缴 N 月当场收款 + 赠送时长阶梯自动命中 + 合约到期月延长(可带券)。

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/promotion"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalPlanRenewReq 续费请求。
type portalPlanRenewReq struct {
	Months    int    `json:"months" binding:"required,gt=0,lte=60"`
	PayMethod string `json:"payMethod" binding:"required"`
	CouponID  string `json:"couponId"` // 可选:缴费抵扣券
}

// portalPlanRenew POST /plans/:planId/renew:购 N 月自动算赠送(promotion 阶梯),
// 缴费落账(可带券)→ 延长合约到期月 → 赠送落痕。
func portalPlanRenew(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		planID, err := strconv.ParseInt(c.Param("planId"), 10, 64)
		if err != nil || planID <= 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		var req portalPlanRenewReq
		if !httpx.BindBody(c, &req) {
			return
		}
		ctx := c.Request.Context()
		productID, fee, err := a.UserData.GetPlanForRenewal(ctx, planID, cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		payNo, receipt, ok := renewRecordPayment(c, a, cid, req, fee)
		if !ok {
			return
		}
		gift := renewGiftMonths(ctx, a, productID, int64(req.Months), cid, receipt.PaymentID)
		a.ResumeAfterPayment(ctx, cid)
		end, err := a.UserData.RenewPlan(ctx, planID, cid, req.Months+gift)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"payNo": payNo, "amount": receipt.Amount, "deductedCents": receipt.DeductedCents,
			"buyMonths": req.Months, "giftMonths": gift, "contractEnd": end,
		})
	}
}

// renewRecordPayment 续费收款:N 月 × 基础月费,可带券同事务核销;失败已回写响应。
func renewRecordPayment(c *gin.Context, a *app.Application, cid int64, req portalPlanRenewReq, fee float64) (string, billing.PaymentReceipt, bool) {
	payNo, err := a.Portal.NextNo(c.Request.Context(), "PAY")
	if err != nil {
		respondErr(c, err)
		return "", billing.PaymentReceipt{}, false
	}
	receipt, err := a.Billing.RecordPaymentWithCoupon(c.Request.Context(), billing.Payment{
		PayNo: payNo, CustomerID: cid, Amount: fee * float64(req.Months),
		Method: req.PayMethod, CouponID: req.CouponID,
	})
	if err != nil {
		respondErr(c, err)
		return "", billing.PaymentReceipt{}, false
	}
	return payNo, receipt, true
}

// renewGiftMonths 赠送阶梯命中与落痕;未命中/落痕失败不影响主流程(返回 0)。
func renewGiftMonths(ctx context.Context, a *app.Application, productID int64, months, cid, paymentID int64) int {
	rule, err := a.Promotion.MatchGiftRule(ctx, productID, int(months))
	if err != nil || rule == nil {
		return 0
	}
	rec := promotion.GiftRecord{
		RuleID: rule.RuleID, CustomerID: cid, ProductID: productID,
		BuyMonths: int(months), GiftMonths: rule.GiftMonths, PaymentID: paymentID,
	}
	if err := a.Promotion.RecordGift(ctx, rec); err != nil {
		return 0
	}
	return rule.GiftMonths
}
