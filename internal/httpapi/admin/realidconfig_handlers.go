package adminapi

// 实名核验配置路由 handler 实现(承接 registerRealIDConfigRoutes)。
// 形态:GET 全量(掩码)/PUT 分组更新/POST channel 自检(完整性或真实试核)。
// 存储复用 biz_params,secret AES-GCM 加密;运行时通道由 app 装配的 realid.Dynamic 消费(60s 热生效)。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// realidConfigGetHandler GET /realid-config:实名核验全量配置(secret 只回 hasValue 标记)。
func realidConfigGetHandler(a *app.Application) gin.HandlerFunc {
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
		for _, f := range realidFields {
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

// realidConfigPutHandler PUT /realid-config/{group}:实名核验分组部分更新。
func realidConfigPutHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		group := c.Param("group")
		if len(realidFieldByGroup(group)) == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		var req struct {
			Values map[string]string `json:"values" binding:"required"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if !realidKeysInGroup(req.Values, group) {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if !realidSaveGroup(c, a, group, req.Values) {
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// realidConfigTestHandler POST /realid-config/channel/test:实名核验自检(无 name/idNo=完整性;带=真实试核)。
func realidConfigTestHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Values map[string]string `json:"values"`
			Name   string            `json:"name"`
			IdNo   string            `json:"idNo"`
		}
		_ = c.ShouldBindJSON(&req)
		cur, err := realidMergedParams(c, a, req.Values, "channel")
		if err != nil {
			respondErr(c, err)
			return
		}
		if req.Name == "" || req.IdNo == "" {
			respond(c, apitypes.CodeOK, realidCompletenessResult(cur))
			return
		}
		start := time.Now()
		decision, err := realidVerifierFromParams(cur).Verify(c.Request.Context(), req.Name, req.IdNo)
		respond(c, apitypes.CodeOK, gin.H{
			"ok": err == nil, "latencyMs": time.Since(start).Milliseconds(),
			"message": realidTestMessage(decision, err),
		})
	}
}