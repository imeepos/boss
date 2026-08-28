package workerapi

// 师傅端现场收款(独立文件控制 scan.go 行数红线):
// 预取应收(与环节4 收款同口径)、payMethods 随 Stripe 通道配置动态下发、落账 payments 并回 payNo。

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerChargeGetHandler 现场收款预取:预付费订单应收(与环节4 收款同口径:月数x月费,区域覆盖优先);
// 可收方式由 Stripe 通道配置决定——已配置含线上卡收款(CARD),未配置默认仅线下(现金/扫码/POS)。
func workerChargeGetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, ord, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		methods := []string{"CASH", "QR", "POS"}
		if a.StripeReady(c.Request.Context()) {
			methods = append([]string{"CARD"}, methods...)
		}
		amountDue, months, prepaid, err := a.Order.PrepaidAmount(c.Request.Context(), ord.ID)
		if err != nil {
			respondErr(c, err)
			return
		}
		desc := ""
		if !prepaid {
			amountDue = 0 // 后付费无强制应收,现场实收自填
			desc = postpaidChargeDesc
		} else {
			desc = fmt.Sprintf("预缴 %d 个月", months)
		}
		respond(c, apitypes.CodeOK, gin.H{
			"ticketNo": tk.TicketNo, "amountDue": amountDue, "amountDesc": desc,
			"payMethods": methods,
		})
	}
}

// postpaidChargeDesc 后付费订单现场收款提示(amountDue=0 时展示给师傅)。
const postpaidChargeDesc = "后付费订单,实收金额自填"

// workerChargeReq 现场收款确认。
type workerChargeReq struct {
	Amount    float64 `json:"amount" binding:"required"`
	PayMethod string  `json:"payMethod" binding:"required"`
}

// validWorkerPayMethod 现场收款方式白名单(线下 CASH/QR/POS + 线上 CARD)。
func validWorkerPayMethod(m string) bool {
	switch m {
	case "CARD", "CASH", "QR", "POS":
		return true
	}
	return false
}

// workerPayMethodToPayment 现场收款方式 → payments.method 枚举(CARD→card 线上;其余→offline 线下)。
func workerPayMethodToPayment(m string) string {
	if m == "CARD" {
		return "card"
	}
	return "offline"
}

// workerChargePostHandler 现场收款落账:pay_no 派单 + payments 流水(bill 为空 + customer_id 归属,
// 与环节4 预付费当场收款同口径);成功回 payNo/receiptUrl,落账失败留 FAILED 可 grep 日志。
func workerChargePostHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, ord, err := ticketOrder(c, a)
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
			if !validWorkerPayMethod(req.PayMethod) {
				return &httpx.ValidationError{Field: "payMethod", Message: "must be one of CARD/CASH/QR/POS"}
			}
			return nil
		}) {
			return
		}
		if a.Billing == nil {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		payNo, err := a.Portal.NextNo(c.Request.Context(), "PAY")
		if err != nil {
			respondErr(c, err)
			return
		}
		if _, err := a.Billing.CreatePayment(c.Request.Context(), billing.Payment{
			PayNo: payNo, CustomerID: ord.CustomerID, Amount: req.Amount,
			Method: workerPayMethodToPayment(req.PayMethod), Status: "SUCCESS",
		}); err != nil {
			log.Printf("[worker-charge] RECORD FAILED ticket=%s order=%d amount=%.2f payMethod=%s err=%v",
				c.Param("ticketNo"), ord.ID, req.Amount, req.PayMethod, err)
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "worker_charge", c.Param("ticketNo"), map[string]any{
			"amount": req.Amount, "payMethod": req.PayMethod, "payNo": payNo,
		})
		respond(c, apitypes.CodeOK, gin.H{
			"ok": true, "payNo": payNo,
			"receiptUrl": "/api/user/v1/payments/" + payNo + "/receipt",
		})
	}
}
