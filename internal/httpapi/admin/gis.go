package adminapi

// 阶段8 GIS 路由:八级下钻 / 全级数量统计 / 资产实时详情(承接 admin gis.html,menu:gis)。
// handler 实现在 gis_handlers.go。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerGisRoutes 注册 GIS 数字孪生路由。
func registerGisRoutes(g *gin.RouterGroup, a *app.Application) {
	gi := g.Group("", requirePerm(a.User, "menu:gis"))
	gi.GET("/gis/drill", gisDrill(a))
	gi.GET("/gis/levels", gisLevelCounts(a))
	gi.GET("/gis/points", gisPoints(a))
	gi.GET("/gis/odn-points", gisODNPoints(a))
	gi.GET("/gis/resources/:resourceId/detail", gisResourceDetail(a))
}
