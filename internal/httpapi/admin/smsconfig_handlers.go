package adminapi

// 短信配置路由 handler 实现(承接 registerSMSConfigRoutes)。
// 形态:GET 全量(掩码)/PUT 分组更新/POST channel 自检(完整性或真实试发)。
// 存储复用 biz_params,secret AES-GCM 加密;运行时通道由 app 装配的 sms.Dynamic 消费(60s 热生效)。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// smsConfigGetHandler GET /sms-config:短信全量配置(secret 只回 hasValue 标记)。
func smsConfigGetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.User.ListParams(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		stored := map[string]string{}
		for _, p := range list {
			stored[p.Key] = p.Value
		}
		fields := gin.H{}
		for _, f := range smsFields {
			v := stored[f.Key]
			if f.Secret {
				fields[f.Key] = gin.H{"value": "", "hasValue": v != ""}
				continue
			}
			if v == "" {
				v = f.Default
			}
			fields[f.Key] = gin.H{"value": v, "hasValue": v != ""}
		}
		respond(c, apitypes.CodeOK, gin.H{"fields": fields})
	}
}

// smsConfigPutHandler PUT /sms-config/{group}:短信分组部分更新。
func smsConfigPutHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		group := c.Param("group")
		if len(smsFieldByGroup(group)) == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		var req struct {
			Values map[string]string `json:"values" binding:"required"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if !smsKeysInGroup(req.Values, group) {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if !smsSaveGroup(c, a, group, req.Values) {
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// smsConfigTestHandler POST /sms-config/channel/test:短信 channel 自检(无 phone=完整性;带 phone=真实试发)。
func smsConfigTestHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Values map[string]string `json:"values"`
			Phone  string            `json:"phone"`
		}
		_ = c.ShouldBindJSON(&req)
		cur, err := smsMergedParams(c, a, req.Values, "channel")
		if err != nil {
			respondErr(c, err)
			return
		}
		if req.Phone == "" {
			respond(c, apitypes.CodeOK, smsCompletenessResult(cur))
			return
		}
		start := time.Now()
		code, _ := randDigitsStr(6)
		err = smsSenderFromParams(cur).Send(c.Request.Context(), req.Phone, code, "test")
		respond(c, apitypes.CodeOK, gin.H{
			"ok": err == nil, "latencyMs": time.Since(start).Milliseconds(),
			"message": smsTestMessage(err),
		})
	}
}
