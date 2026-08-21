package adminapi

// 实名核验配置路由:GET 全量(掩码)/PUT 分组更新/POST channel 自检(完整性或真实试核)。
// 存储复用 biz_params,secret AES-GCM 加密;运行时通道由 app 装配的 realid.Dynamic 消费(60s 热生效)。
// 未启用/未配置 → 提交落 PENDING,人工核验兜底(customer_onboarding 的 real-name/verify 不受影响)。

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/realid"
	"github.com/ymm-001/boss/internal/pkg/secretbox"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func registerRealIDConfigRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:realidconfig")

	// 全量配置:secret 只回 hasValue 标记。
	g.GET("/realid-config", perm, func(c *gin.Context) {
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
	})

	// 分组部分更新:secret 空串=不修改;越组 key 拒绝。
	g.PUT("/realid-config/:group", perm, func(c *gin.Context) {
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
	})

	// channel 自检:无 name/idNo=配置完整性校验;带 =用草稿合并配置真实试核一组二要素。
	g.POST("/realid-config/channel/test", perm, func(c *gin.Context) {
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
	})
}

// realidSaveGroup 逐字段落库 + 审计;secret 加密,空串跳过。
func realidSaveGroup(c *gin.Context, a *app.Application, group string, values map[string]string) bool {
	accountID := httpx.ClaimsAccountID(c)
	for _, f := range realidFieldByGroup(group) {
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
		httpx.RecordAudit(a, c, "数据变更", "realid_config", f.Key, authAuditDetail(f, raw))
	}
	return true
}

// realidKeysInGroup 全部 key 属于该组才放行。
func realidKeysInGroup(values map[string]string, group string) bool {
	for key := range values {
		f, ok := realidFieldByKey(key)
		if !ok || f.Group != group {
			return false
		}
	}
	return true
}

// realidMergedParams 已存 biz_params(含默认值)叠加草稿 secret 解密后合并。
func realidMergedParams(c *gin.Context, a *app.Application, draft map[string]string, group string) (map[string]string, error) {
	list, err := a.User.ListParams(c.Request.Context())
	if err != nil {
		return nil, err
	}
	cur := map[string]string{}
	for _, p := range list {
		cur[p.Key] = p.Value
	}
	for _, f := range realidFields {
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
		if f, ok := realidFieldByKey(k); ok && f.Group == group && !(f.Secret && v == "") {
			cur[k] = v
		}
	}
	return cur, nil
}

// realidCompletenessResult 配置完整性:未启用视为通过;缺必填返回明细。
func realidCompletenessResult(cur map[string]string) gin.H {
	if cur["realid.enabled"] == "false" {
		return gin.H{"ok": true, "latencyMs": 0, "message": "通道未启用,实名提交保持人工核验"}
	}
	var missing []string
	for _, key := range realidTestRequired {
		if cur[key] == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return gin.H{"ok": false, "latencyMs": 0, "message": "缺少必填字段: " + joinKeys(missing)}
	}
	return gin.H{"ok": true, "latencyMs": 0, "message": "配置完整性校验通过"}
}

// realidVerifierFromParams 合并配置 → 通道实例(试核用;未配置凭据返回 nil)。
func realidVerifierFromParams(cur map[string]string) realid.Verifier {
	return realid.NewAliyunCloudauth(realid.AliyunCloudauthConfig{
		AccessKeyID:     cur["realid.accessKeyId"],
		AccessKeySecret: cur["realid.accessKeySecret"],
		Endpoint:        cur["realid.endpoint"],
	})
}

// realidTestMessage 试核结果文案。
func realidTestMessage(decision string, err error) string {
	if err != nil {
		if errors.Is(err, realid.ErrDisabled) {
			return "通道未启用或凭据缺失"
		}
		return "核验失败: " + err.Error()
	}
	if decision == realid.Pass {
		return "二要素一致(PASS)"
	}
	return "二要素不一致/查无(" + decision + ")"
}
