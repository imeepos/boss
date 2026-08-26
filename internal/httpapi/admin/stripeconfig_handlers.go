package adminapi

// Stripe 支付配置路由 handler 实现(承接 registerStripeConfigRoutes)。
// 形态:GET 全量(掩码)/PUT 分组更新/POST channel 自检(完整性或真实余额探活)。
// 存储复用 biz_params,secret AES-GCM 加密;运行时通道由 app 装配的 stripe.Dynamic 消费(60s 热生效)。

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/secretbox"
	"github.com/ymm-001/boss/internal/pkg/stripe"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// stripeConfigGetHandler GET /stripe-config:支付通道全量配置(secret 只回 hasValue 标记)。
func stripeConfigGetHandler(a *app.Application) gin.HandlerFunc {
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
		for _, f := range stripeFields {
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

// stripeConfigPutHandler PUT /stripe-config/{group}:支付分组部分更新(channel/webhook)。
func stripeConfigPutHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		group := c.Param("group")
		if len(stripeFieldByGroup(group)) == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		var req struct {
			Values map[string]string `json:"values" binding:"required"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if !stripeKeysInGroup(req.Values, group) {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if !stripeSaveGroup(c, a, group, req.Values) {
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// stripeProbe 自检探活函数(测试可注入;默认真实 Stripe 余额探活,零副作用)。
var stripeProbe = stripeProbeBalance

// stripeConfigTestHandler POST /stripe-config/channel/test:通道自检
// (完整性校验跨 channel/webhook 两组;完整性通过后以草稿+已存配置真实探活余额)。
func stripeConfigTestHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Values map[string]string `json:"values"`
		}
		_ = c.ShouldBindJSON(&req)
		cur, err := stripeMergedParams(c, a, req.Values, "channel")
		if err != nil {
			respondErr(c, err)
			return
		}
		if cur["stripe.enabled"] == "false" {
			respond(c, apitypes.CodeOK, gin.H{"ok": true, "latencyMs": 0, "message": "通道未启用,跳过自检"})
			return
		}
		if got := stripeCompletenessResult(cur); got["ok"] != true {
			respond(c, apitypes.CodeOK, got)
			return
		}
		start := time.Now()
		err = stripeProbe(c.Request.Context(), cur)
		if err != nil {
			respond(c, apitypes.CodeOK, gin.H{
				"ok": false, "latencyMs": time.Since(start).Milliseconds(),
				"message": "余额探活失败: " + err.Error(),
			})
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"ok": true, "latencyMs": time.Since(start).Milliseconds(), "message": "配置完整,余额探活成功",
		})
	}
}

// stripeKeysInGroup 全部 key 属于该组才放行。
func stripeKeysInGroup(values map[string]string, group string) bool {
	for key := range values {
		f, ok := stripeFieldByKey(key)
		if !ok || f.Group != group {
			return false
		}
	}
	return true
}

// stripeMergedParams 已存 biz_params(含默认值)叠加草稿 secret 解密后合并。
func stripeMergedParams(c *gin.Context, a *app.Application, draft map[string]string, group string) (map[string]string, error) {
	list, err := a.User.ListParams(c.Request.Context())
	if err != nil {
		return nil, err
	}
	cur := map[string]string{}
	for _, p := range list {
		cur[p.Key] = p.Value
	}
	for _, f := range stripeFields {
		if cur[f.Key] == "" {
			cur[f.Key] = f.Default
		}
		if f.Secret && cur[f.Key] != "" {
			if plain, err := secretbox.Open(cur[f.Key]); err == nil {
				cur[f.Key] = plain
			}
		}
	}
	for k, v := range draft {
		if f, ok := stripeFieldByKey(k); ok && f.Group == group && !(f.Secret && v == "") {
			cur[k] = v
		}
	}
	return cur, nil
}

// stripeCompletenessResult 配置完整性:缺必填返回明细(未启用由调用方提前返回)。
func stripeCompletenessResult(cur map[string]string) gin.H {
	var missing []string
	for _, key := range stripeTestRequired {
		if cur[key] == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return gin.H{"ok": false, "latencyMs": 0, "message": "缺少必填字段: " + joinKeys(missing)}
	}
	return gin.H{"ok": true, "latencyMs": 0, "message": "配置完整性校验通过"}
}

// stripeProbeBalance 以合并配置构造客户端探活 /v1/balance(零副作用)。
func stripeProbeBalance(ctx context.Context, cur map[string]string) error {
	client := stripe.New(cur["stripe.apiKey"], cur["stripe.currency"])
	if client == nil {
		return stripe.ErrDisabled
	}
	if base := cur["stripe.apiBaseUrl"]; base != "" {
		client.BaseURL = base
	}
	return client.ProbeBalance(ctx)
}
