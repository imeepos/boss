package adminapi

// Stripe 支付配置路由 handler 实现(承接 registerStripeConfigRoutes)。
// 形态:GET 全量(掩码)/PUT 分组更新/POST channel 自检(完整性或真实余额探活)。
// 存储复用 biz_params,secret AES-GCM 加密;运行时通道由 app 装配的 stripe.Dynamic 消费(60s 热生效)。

import (
	"context"
	"strings"
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
// (完整性校验跨 channel/webhook 两组;草稿合并也跨两组——webhookSecret 未保存即可参与;
// 完整性通过后以草稿+已存配置真实探活余额,并核对 Stripe 后台 webhook endpoint 一致性)。
func stripeConfigTestHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Values map[string]string `json:"values"`
		}
		_ = c.ShouldBindJSON(&req)
		cur, err := stripeMergedParams(c, a, req.Values)
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
		if err = stripeProbe(c.Request.Context(), cur); err != nil {
			respond(c, apitypes.CodeOK, gin.H{
				"ok": false, "latencyMs": time.Since(start).Milliseconds(),
				"message": "余额探活失败: " + err.Error(),
			})
			return
		}
		msg := stripeEndpointCheckFunc(c.Request.Context(), cur)
		respond(c, apitypes.CodeOK, gin.H{
			"ok": true, "latencyMs": time.Since(start).Milliseconds(),
			"message": "配置完整,余额探活成功" + msg,
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

// stripeMergedParams 已存 biz_params(含默认值)叠加草稿(跨全部组)解密后合并。
func stripeMergedParams(c *gin.Context, a *app.Application, draft map[string]string) (map[string]string, error) {
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
		if f, ok := stripeFieldByKey(k); ok && !(f.Secret && v == "") {
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

// stripeEndpointCheckFunc 后台 endpoint 一致性核对(测试可注入;默认真实查询,零副作用)。
var stripeEndpointCheckFunc = stripeEndpointCheckImpl

// stripeEndpointCheckImpl 核对 Stripe 后台 webhook endpoint 与期望 URL 一致性(只读,零副作用)。
// 返回拼接进自检 message 的说明:URL 失配/缺失/期望未配置都会明确提示(隧道变化后静默失效的
// 根因——endpoint 重建导致 whsec 过期,URL 检查是唯一可经 API 观测的代理指标)。
func stripeEndpointCheckImpl(ctx context.Context, cur map[string]string) string {
	want := strings.TrimRight(cur["stripe.webhookUrl"], "/")
	if want == "" {
		return "。提示:未配置期望回调 URL(回调卡片 webhookUrl),隧道变化时无法自愈/自检比对"
	}
	client := stripe.New(cur["stripe.apiKey"], cur["stripe.currency"])
	if client == nil {
		return ""
	}
	if base := cur["stripe.apiBaseUrl"]; base != "" {
		client.BaseURL = base
	}
	endpoints, err := client.ListWebhookEndpoints(ctx)
	if err != nil {
		return "。提示:查询 Stripe webhook endpoint 失败: " + err.Error()
	}
	for _, ep := range endpoints {
		if !strings.Contains(ep.URL, "/api/user/v1/webhooks/stripe") {
			continue
		}
		if strings.TrimRight(ep.URL, "/") == want {
			return "。后台 webhook endpoint 与期望 URL 一致"
		}
		return "。警告:后台 webhook endpoint URL(" + ep.URL + ")与期望不一致(" + want + "),回调将静默失效,自愈循环会自动同步"
	}
	return "。警告:后台未注册本系统 webhook endpoint,回调将静默失效,自愈循环会自动重建"
}
