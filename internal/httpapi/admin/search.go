package adminapi

// 聚合搜索路由:GET /search?keyword= 四域统一检索(契约 api/openapi/admin/search.yaml)。
// 权限不在路由层 gate(账号持有的域权限各不相同),由 handler 按域逐一判定。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerSearchRoutes 注册聚合搜索路由(仅 Authn;域权限在 handler 内过滤)。
func registerSearchRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/search", searchAggregateHandler(a))
}
