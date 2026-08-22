// ai 路由:OpenAI 能力统一接口 + admin 集中配置(menu:ai 门禁,初始仅 sysadmin)。
package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerAIRoutes 注册 AI 能力路由:配置读写 + chat/embeddings 调用。
func registerAIRoutes(g *gin.RouterGroup, a *app.Application) {
	ag := g.Group("", requirePerm(a.User, "menu:ai"))
	ag.GET("/ai/openai/config", aiGetConfigHandler(a))
	ag.PUT("/ai/openai/config", aiUpdateConfigHandler(a))
	ag.POST("/ai/chat/completions", aiChatHandler(a))
	ag.POST("/ai/embeddings", aiEmbeddingHandler(a))
}
