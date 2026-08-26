package adminapi

// Stripe 支付配置路由:GET 全量(掩码)/PUT 分组更新(channel/webhook)/POST channel 自检。
// 存储复用 biz_params,secret AES-GCM 加密;运行时通道由 app 装配的 stripe.Dynamic 消费(60s 热生效)。
// 全部 handler 实现见 stripeconfig_handlers.go;此处只保留扁平路由表 + 落库辅助。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerStripeConfigRoutes 注册支付配置路由。
func registerStripeConfigRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:stripeconfig")

	g.GET("/stripe-config", perm, stripeConfigGetHandler(a))
	g.PUT("/stripe-config/:group", perm, stripeConfigPutHandler(a))
	g.POST("/stripe-config/channel/test", perm, stripeConfigTestHandler(a))
}

// stripeSaveGroup 逐字段落库 + 审计;secret 加密,空串跳过(掩码回显未修改)。
func stripeSaveGroup(c *gin.Context, a *app.Application, group string, values map[string]string) bool {
	accountID := httpx.ClaimsAccountID(c)
	for _, f := range stripeFieldByGroup(group) {
		raw, ok := values[f.Key]
		if !ok {
			continue
		}
		if f.Secret && raw == "" {
			continue
		}
		val := raw
		if f.Secret {
			enc, err := sealParamSecret(raw)
			if err != nil {
				respond(c, apitypes.CodeInternal, nil)
				return false
			}
			val = enc
		}
		if err := a.User.UpdateParam(c.Request.Context(), f.Key, val, accountID); err != nil {
			respondErr(c, err)
			return false
		}
		httpx.RecordAudit(a, c, "数据变更", "stripe_config", f.Key, authAuditDetail(f, raw))
	}
	return true
}
