package adminapi

// 短信配置路由:GET 全量(掩码)/PUT 分组更新/POST channel 自检(完整性或真实试发)。
// 存储复用 biz_params,secret AES-GCM 加密;运行时通道由 app 装配的 sms.Dynamic 消费(60s 热生效)。
// 全部 handler 实现见 config_handlers.go;此处只保留扁平路由表。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerSMSConfigRoutes 注册短信配置路由。
func registerSMSConfigRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:smsconfig")

	g.GET("/sms-config", perm, smsConfigGetHandler(a))
	g.PUT("/sms-config/:group", perm, smsConfigPutHandler(a))
	g.POST("/sms-config/channel/test", perm, smsConfigTestHandler(a))
}

// smsSaveGroup 逐字段落库 + 审计;secret 加密,空串跳过。
func smsSaveGroup(c *gin.Context, a *app.Application, group string, values map[string]string) bool {
	accountID := httpx.ClaimsAccountID(c)
	for _, f := range smsFieldByGroup(group) {
		raw, ok := values[f.Key]
		if !ok {
			continue
		}
		if f.Secret && raw == "" {
			continue // 掩码回显未修改:跳过
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
		httpx.RecordAudit(a, c, "数据变更", "sms_config", f.Key, authAuditDetail(f, raw))
	}
	return true
}
