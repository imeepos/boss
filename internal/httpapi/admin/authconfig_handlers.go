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
		for key := range req.Values {
			f, ok := authFieldByKey(key)
			if !ok || f.Group != group {
				respond(c, apitypes.CodeInvalidParam, gin.H{"error": "key does not belong to group"})
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
				enc, err := sealParamSecret(raw)
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
	}
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
	}
}