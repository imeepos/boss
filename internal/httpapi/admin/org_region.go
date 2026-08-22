package adminapi

// 经营区域路由(list + 覆盖主体划分,从 org.go 拆出,保持单文件 ≤300 行)。
// handler 实现在 org_region_handlers.go。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerRegionRoutes 经营区域路由(g 组)。
func registerRegionRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/regions", requirePerm(a.User, "menu:region"), regionList(a))
	// 区域覆盖主体划分:子公司划经营区域/摘除(migrations/000076;0=摘除回落祖先/总公司兜底)。
	g.PUT("/regions/:regionId/coverage", requirePerm(a.User, "menu:region"), regionAssignCoverage(a))
}
