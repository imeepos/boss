// ai 路由:OpenAI 能力统一接口 + admin 集中配置(menu:ai 门禁,初始仅 sysadmin)。
package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/ai"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerAIRoutes 注册 AI 能力路由:配置读写 + chat/embeddings 调用。
func registerAIRoutes(g *gin.RouterGroup, a *app.Application) {
	ag := g.Group("", requirePerm(a.User, "menu:ai"))

	ag.GET("/ai/openai/config", func(c *gin.Context) {
		view, err := a.AI.ConfigView(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, view)
	})

	ag.PUT("/ai/openai/config", func(c *gin.Context) {
		var req ai.ConfigUpdate
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		claims, _ := c.Get(middleware.CtxClaims)
		cl, _ := claims.(*auth.Claims)
		var updatedBy int64
		if cl != nil {
			updatedBy = cl.AccountID
		}
		if err := a.AI.UpdateConfig(c.Request.Context(), req, updatedBy); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "ai_openai_config", "biz_params", gin.H{
			"apiUrlSet": req.APIURL != nil, "apiKeySet": req.APIKey != nil, "modelSet": req.Model != nil,
		})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	ag.POST("/ai/chat/completions", func(c *gin.Context) {
		var req ai.ChatRequest
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		res, err := a.AI.ChatCompletion(c.Request.Context(), req)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, res)
	})

	ag.POST("/ai/embeddings", func(c *gin.Context) {
		var req ai.EmbeddingRequest
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		res, err := a.AI.Embeddings(c.Request.Context(), req)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, res)
	})
}
