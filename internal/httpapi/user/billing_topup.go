package userapi

// 用户端充值(余额入账)handler:portalBalanceGet / portalTopup,
// 合成客户(隔离空间负数 ID)cid<=0 一律拒(adopted 2026-09-03)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalTopupReq 充值请求体。
type portalTopupReq struct {
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	PayMethod string  `json:"payMethod" binding:"required"`
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
//
// 合成客户(隔离空间负数 ID)拒绝充值:`payments.customer_id` 是硬 FK,
// 负数 ID 在 customers 表无对应行会触发 23503;合成客户不在真实收费场景。
func portalTopup(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		if cid <= 0 {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "合成客户不支持充值"})
			return
		}
		var req portalTopupReq
		if !httpx.BindBody(c, &req) {
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

// topupRecordPayment 充值派单号 + 原子落账(bill_id NULL + customer_id 归属,
// RecordTopup 同事务增余额;否则 /payments 与凭证端点查不到);失败已回写响应。
func topupRecordPayment(c *gin.Context, a *app.Application, cid int64, req portalTopupReq) (string, bool) {
	payNo, err := a.Portal.NextNo(c.Request.Context(), "PAY")
	if err != nil {
		respondErr(c, err)
		return "", false
	}
	if _, err := a.Billing.RecordTopup(c.Request.Context(), billing.Payment{
		PayNo: payNo, CustomerID: cid, Amount: req.Amount,
		Method: req.PayMethod, Status: "SUCCESS",
	}); err != nil {
		respondErr(c, err)
		return "", false
	}
	return payNo, true
}