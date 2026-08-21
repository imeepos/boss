package adminapi

// 推送配置路由:GET 全量(掩码)/PUT 分组更新/POST channel 自检(完整性或真实试发)。
// 存储复用 biz_params,secret AES-GCM 加密;运行时通道由 app 装配的 push.Dynamic 消费(60s 热生效)。
// 设计契约:docs/plan/push-config-design.md。

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/push"
	"github.com/ymm-001/boss/internal/pkg/secretbox"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func registerPushConfigRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:pushconfig")

	// 全量配置:secret 只回 hasValue 标记。
	g.GET("/push-config", perm, func(c *gin.Context) {
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
	})

	// 分组部分更新:secret 空串=不修改;越组 key 拒绝。
	g.PUT("/push-config/:group", perm, func(c *gin.Context) {
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
	})

	// channel 自检:无 target=配置完整性校验;带 target(registrationID|alias)=真实试发一条测试通知。
	g.POST("/push-config/channel/test", perm, func(c *gin.Context) {
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
	})
}

// pushSaveGroup 逐字段落库 + 审计;secret 加密,空串跳过。
func pushSaveGroup(c *gin.Context, a *app.Application, group string, values map[string]string) bool {
	accountID := httpx.ClaimsAccountID(c)
	for _, f := range pushFieldByGroup(group) {
		raw, ok := values[f.Key]
		if !ok {
			continue
		}
		if f.Secret && raw == "" {
			continue // 掩码回显未修改:跳过
		}
		val := raw
		if f.Secret {
			enc, err := secretbox.Seal(raw)
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
		httpx.RecordAudit(a, c, "数据变更", "push_config", f.Key, authAuditDetail(f, raw))
	}
	return true
}

// pushKeysInGroup 全部 key 属于该组才放行。
func pushKeysInGroup(values map[string]string, group string) bool {
	for key := range values {
		f, ok := pushFieldByKey(key)
		if !ok || f.Group != group {
			return false
		}
	}
	return true
}

// pushMergedParams 已存 biz_params(含默认值)叠加草稿 secret 解密后合并。
func pushMergedParams(c *gin.Context, a *app.Application, draft map[string]string, group string) (map[string]string, error) {
	list, err := a.User.ListParams(c.Request.Context())
	if err != nil {
		return nil, err
	}
	cur := map[string]string{}
	for _, p := range list {
		cur[p.Key] = p.Value
	}
	for _, f := range pushFields {
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
		if f, ok := pushFieldByKey(k); ok && f.Group == group && !(f.Secret && v == "") {
			cur[k] = v
		}
	}
	return cur, nil
}

// pushCompletenessResult 配置完整性:未启用视为通过;缺必填返回明细。
func pushCompletenessResult(cur map[string]string) gin.H {
	if cur["push.enabled"] == "false" {
		return gin.H{"ok": true, "latencyMs": 0, "message": "通道未启用,跳过自检"}
	}
	var missing []string
	for _, key := range pushTestRequired {
		if cur[key] == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return gin.H{"ok": false, "latencyMs": 0, "message": "缺少必填字段: " + joinKeys(missing)}
	}
	return gin.H{"ok": true, "latencyMs": 0, "message": "配置完整性校验通过"}
}

// pushSenderFromParams 合并配置 → 真实试发(自检用);返回消息 ID。
func pushSenderFromParams(ctx context.Context, cur map[string]string, target, targetKind string) (string, error) {
	live, _ := strconv.ParseInt(cur["push.jpush.liveTime"], 10, 64)
	cfg := push.ChannelConfig{
		Enabled:        cur["push.enabled"] != "false",
		AppKey:         cur["push.jpush.appKey"],
		MasterSecret:   cur["push.jpush.masterSecret"],
		APIURL:         cur["push.jpush.apiUrl"],
		ApnsProduction: cur["push.jpush.apnsProduction"] != "false",
		LiveTimeSec:    live,
	}
	if !cfg.Enabled {
		return "", push.ErrDisabled
	}
	if cfg.AppKey == "" || cfg.MasterSecret == "" {
		return "", errors.New("推送凭据未配置")
	}
	req := push.Request{Title: "BOSS", Alert: "推送通道测试通知"}
	if targetKind == "alias" {
		req.Alias = []string{target}
	} else {
		req.RegistrationIDs = []string{target}
	}
	return push.NewJPush(cfg, nil).Send(ctx, req)
}

// pushTestMessage 试发结果文案。
func pushTestMessage(err error, msgID string) string {
	if err == nil {
		return "测试通知已提交发送(msg_id=" + msgID + ")"
	}
	if errors.Is(err, push.ErrDisabled) {
		return "通道未启用"
	}
	return "发送失败: " + err.Error()
}
