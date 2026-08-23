package adminapi

// 号码认证配置路由(auth-config-v1.spec.md §2):GET 全量(掩码)/PUT 分组部分更新/POST 自检。
// 存储复用 biz_params(User.ListParams/UpdateParam),secret 字段 AES-GCM 加密后落库。
// 全部 handler 实现见 config_handlers.go;此处只保留扁平路由表 + 共享审计/自检辅助。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerAuthConfigRoutes 注册号码认证配置路由。
func registerAuthConfigRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:authconfig")

	g.GET("/auth-config", perm, authConfigGetHandler(a))
	g.PUT("/auth-config/:group", perm, authConfigPutHandler(a))
	g.POST("/auth-config/:group/test", perm, authConfigTestHandler(a))
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
