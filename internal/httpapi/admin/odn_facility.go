package adminapi

// ODN 设施/光缆/纤芯路由注册(承接 registerODN*)。
// 全部 handler 实现见 odn_facility_handlers.go;此处只保留扁平路由表。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerODNFacilityRoutes 设施路由(列表/详情/新建/报废)。
func registerODNFacilityRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.GET("/odn/facilities", perm, odnListFacilitiesHandler(a))
	g.GET("/odn/facilities/:code", perm, odnGetFacilityHandler(a))
	g.POST("/odn/facilities", perm, odnCreateFacilityHandler(a))
	g.DELETE("/odn/facilities/:code", perm, odnRetireFacilityHandler(a))
}

// registerODNCableRoutes 光缆段落/纤芯路由。
func registerODNCableRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.GET("/odn/segments", perm, odnListSegmentsHandler(a))
	g.POST("/odn/segments", perm, odnCreateSegmentHandler(a))
	g.POST("/odn/segments/:id/fibers", perm, odnAddFiberHandler(a))
	g.GET("/odn/segments/:id/fibers", perm, odnListFibersHandler(a))
}