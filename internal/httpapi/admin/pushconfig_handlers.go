package adminapi

// 推送配置路由 handler 实现(承接 registerPushConfigRoutes)。
// 形态:GET 全量(掩码)/PUT 分组更新/POST channel 自检(完整性或真实试发)。
// 存储复用 biz_params,secret AES-GCM 加密;运行时通道由 app 装配的 push.Dynamic 消费(60s 热生效)。
// 设计契约:docs/plan/push-config-design.md。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// pushConfigGetHandler GET /push-config:推送全量配置(secret 只回 hasValue 标记)。
func pushConfigGetHandler(a *app.Application) gin.HandlerFunc {
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
		for _, f := range pushFields {
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

// pushConfigPutHandler PUT /push-config/{group}:推送分组部分更新。
func pushConfigPutHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		group := c.Param("group")
		if len(pushFieldByGroup(group)) == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		var req struct {
			Values map[string]string `json:"values" binding:"required"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if !pushKeysInGroup(req.Values, group) {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if !pushSaveGroup(c, a, group, req.Values) {
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// pushConfigTestHandler POST /push-config/channel/test:推送自检(无 target=完整性;带 target=真实试发)。
func pushConfigTestHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Values     map[string]string `json:"values"`
			Target     string            `json:"target"`
			TargetKind string            `json:"targetKind"` // registration_id(默认) | alias
		}
		_ = c.ShouldBindJSON(&req)
		cur, err := pushMergedParams(c, a, req.Values, "channel")
		if err != nil {
			respondErr(c, err)
			return
		}
		if req.Target == "" {
			respond(c, apitypes.CodeOK, pushCompletenessResult(cur))
			return
		}
		start := time.Now()
		msgID, err := pushSenderFromParams(c.Request.Context(), cur, req.Target, req.TargetKind)
		respond(c, apitypes.CodeOK, gin.H{
			"ok": err == nil, "latencyMs": time.Since(start).Milliseconds(), "message": pushTestMessage(err, msgID),
		})
	}
}