package userapi

// 支付方式下发:App 端缴费/充值展示不再硬编码,以本接口为准。
// Stripe 通道可用(后台「支付配置」已配且启用)→ 含银行卡线上收款,default=card;
// 未配置 → 不含银行卡,default=cash(线下收款,师傅上门/柜台核收),与"密钥未配即降级"裁定一致。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalPayMethods GET /payments/methods:当前可用支付方式列表。
func portalPayMethods(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		items := []gin.H{
			{"key": "wechat", "label": "微信支付"},
			{"key": "alipay", "label": "支付宝"},
			{"key": "cash", "label": "线下收款"},
		}
		defaultMethod := "cash"
		if a.StripeReady(c.Request.Context()) {
			items = append(items, gin.H{"key": "card", "label": "银行卡"})
			defaultMethod = "card"
		}
		respond(c, apitypes.CodeOK, gin.H{"default": defaultMethod, "items": items})
	}
}
