// ai 路由具名 handler(承接 registerAIRoutes 扁平路由表)。
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

// aiGetConfigHandler GET /ai/openai/config:读取 OpenAI 配置视图。
func aiGetConfigHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		view, err := a.AI.ConfigView(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, view)
	}
}

// aiUpdateConfigHandler PUT /ai/openai/config:更新 OpenAI 配置。
func aiUpdateConfigHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ai.ConfigUpdate
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		updatedBy := aiClaimsAccountID(c)
		if err := a.AI.UpdateConfig(c.Request.Context(), req, updatedBy); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "ai_openai_config", "biz_params", gin.H{
			"apiUrlSet": req.APIURL != nil, "apiKeySet": req.APIKey != nil, "modelSet": req.Model != nil,
		})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// aiClaimsAccountID 读取 JWT 中的 AccountID(API key 主体未参与改 AI 配置)。
func aiClaimsAccountID(c *gin.Context) int64 {
	claims, _ := c.Get(middleware.CtxClaims)
	cl, _ := claims.(*auth.Claims)
	if cl == nil {
		return 0
	}
	return cl.AccountID
}

// aiChatHandler POST /ai/chat/completions:OpenAI 聊天补全。
func aiChatHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// aiEmbeddingHandler POST /ai/embeddings:OpenAI 向量嵌入。
func aiEmbeddingHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}
