package adminapi

import (
	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
)

// registerKnowledgeRoutes 注册客服知识库路由(menu:complaint 权限,复用 CS 域权限码)。
func registerKnowledgeRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/knowledge-articles", requirePerm(a.User, "menu:complaint"), knowledgeListHandler(a))
	g.POST("/knowledge-articles", requirePerm(a.User, "menu:complaint"), knowledgeCreateHandler(a))
	g.PUT("/knowledge-articles/:articleId", requirePerm(a.User, "menu:complaint"), knowledgeUpdateHandler(a))
	g.DELETE("/knowledge-articles/:articleId", requirePerm(a.User, "menu:complaint"), knowledgeDeleteHandler(a))
}
