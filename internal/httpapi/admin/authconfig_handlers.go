package adminapi

// 号码认证配置路由 handler 实现(承接 registerAuthConfigRoutes)。
// 形态:GET 全量(掩码)/PUT 分组更新/POST 自检(配置完整性)。
// 存储复用 biz_params,secret AES-GCM 加密。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// authConfigGetHandler GET /auth-config:号码认证全量配置(secret 只回 hasValue 标记)。
func authConfigGetHandler(a *app.Application) gin.HandlerFunc {
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
	}
}

// authConfigPutHandler PUT /auth-config/{group}:号码认证分组部分更新。
func authConfigPutHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		group := c.Param("group")
		if len(authFieldByGroup(group)) == 0 {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "invalid group"})
			return
		}
		var req struct {
			Values map[string]string `json:"values" binding:"required"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if !authConfigKeysInGroup(c, req.Values, group) {
			return
		}
		if !authConfigApplyGroup(c, a, group, req.Values) {
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// authConfigKeysInGroup 校验请求键都归属当前分组;失败已回写响应。
func authConfigKeysInGroup(c *gin.Context, values map[string]string, group string) bool {
	for key := range values {
		f, ok := authFieldByKey(key)
		if !ok || f.Group != group {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "key does not belong to group"})
			return false
		}
	}
	return true
}

// authConfigApplyGroup 逐字段落库(掩码回显跳过;secret 加密后写)+ 审计;失败已回写响应。
func authConfigApplyGroup(c *gin.Context, a *app.Application, group string, values map[string]string) bool {
	accountID := httpx.ClaimsAccountID(c)
	for _, f := range authFieldByGroup(group) {
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
		httpx.RecordAudit(a, c, "数据变更", "auth_config", f.Key, authAuditDetail(f, raw))
	}
	return true
}

// authConfigTestHandler POST /auth-config/{group}/test:号码认证连通性自检(v1=配置完整性校验)。
func authConfigTestHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		group := c.Param("group")
		if len(authFieldByGroup(group)) == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		var req struct {
			Values map[string]string `json:"values"`
		}
		_ = c.ShouldBindJSON(&req)

		cur, ok := authMergedConfig(c, a, group, req.Values)
		if !ok {
			return
		}
		respond(c, apitypes.CodeOK, authTestResult(group, cur))
	}
}
// authMergedConfig 自检配置合成:库内现值 → 缺省补齐 → 请求覆盖(掩码空值跳过);
// 失败已回写响应。
func authMergedConfig(c *gin.Context, a *app.Application, group string, values map[string]string) (map[string]string, bool) {
	list, err := a.User.ListParams(c.Request.Context())
	if err != nil {
		respondErr(c, err)
		return nil, false
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
	for k, v := range values {
		if f, ok := authFieldByKey(k); ok && f.Group == group && !(f.Secret && v == "") {
			cur[k] = v
		}
	}
	return cur, true
}
