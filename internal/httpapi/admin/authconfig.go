package adminapi

// 号码认证配置路由(auth-config-v1.spec.md §2):GET 全量(掩码)/PUT 分组部分更新/POST 自检。
// 存储复用 biz_params(User.ListParams/UpdateParam),secret 字段 AES-GCM 加密后落库。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/secretbox"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func registerAuthConfigRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:authconfig")

	// 全量配置:secret 只回 hasValue 标记,不回明文/掩码文本。
	g.GET("/auth-config", perm, func(c *gin.Context) {
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
		for _, f := range authFields {
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

	// 分组部分更新:values 为全 key map;secret 空串=不修改;非法组/越组 key 拒绝。
	g.PUT("/auth-config/:group", perm, func(c *gin.Context) {
		group := c.Param("group")
		if len(authFieldByGroup(group)) == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		var req struct {
			Values map[string]string `json:"values" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		for key := range req.Values {
			f, ok := authFieldByKey(key)
			if !ok || f.Group != group {
				respond(c, apitypes.CodeInvalidParam, nil)
				return
			}
		}
		accountID := httpx.ClaimsAccountID(c)
		for _, f := range authFieldByGroup(group) {
			raw, ok := req.Values[f.Key]
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
					return
				}
				val = enc
			}
			if err := a.User.UpdateParam(c.Request.Context(), f.Key, val, accountID); err != nil {
				respondErr(c, err)
				return
			}
			httpx.RecordAudit(a, c, "数据变更", "auth_config", f.Key, authAuditDetail(f, raw))
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	// 连通性自检(v1=配置完整性校验):body 可携带草稿 values 合并已存配置后校验。
	g.POST("/auth-config/:group/test", perm, func(c *gin.Context) {
		group := c.Param("group")
		if len(authFieldByGroup(group)) == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		var req struct {
			Values map[string]string `json:"values"`
		}
		_ = c.ShouldBindJSON(&req)

		list, err := a.User.ListParams(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		cur := map[string]string{}
		for _, p := range list {
			cur[p.Key] = p.Value
		}
		for _, f := range authFields {
			if cur[f.Key] == "" {
				cur[f.Key] = f.Default
			}
		}
		for k, v := range req.Values {
			if f, ok := authFieldByKey(k); ok && f.Group == group && !(f.Secret && v == "") {
				cur[k] = v
			}
		}
		respond(c, apitypes.CodeOK, authTestResult(group, cur))
	})
}

// authAuditDetail 审计明细:secret 只记长度,不落明文。
func authAuditDetail(f authField, raw string) map[string]any {
	if f.Secret {
		return map[string]any{"secretUpdated": raw != "", "length": len(raw)}
	}
	return map[string]any{"value": raw}
}

// authTestResult 自检:enabled=false 视为通过(未启用);缺必填字段返回明细。
func authTestResult(group string, cur map[string]string) gin.H {
	if cur["auth."+group+".enabled"] == "false" && group != "fallback" {
		return gin.H{"ok": true, "latencyMs": 0, "message": "未启用,跳过自检"}
	}
	if group == "my" && cur["auth.my.provider"] == "none" {
		return gin.H{"ok": true, "latencyMs": 0, "message": "认证渠道为 none(仅短信),跳过自检"}
	}
	var missing []string
	for _, key := range authTestRequired[group] {
		if cur[key] == "" {
			f, _ := authFieldByKey(key)
			missing = append(missing, f.Key)
		}
	}
	if len(missing) > 0 {
		return gin.H{"ok": false, "latencyMs": 0, "message": "缺少必填字段: " + joinKeys(missing)}
	}
	return gin.H{"ok": true, "latencyMs": 0, "message": "配置完整性校验通过"}
}

func joinKeys(keys []string) string {
	out := ""
	for i, k := range keys {
		if i > 0 {
			out += ", "
		}
		out += k
	}
	return out
}
