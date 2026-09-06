package adminapi

// ODN 覆盖关联路由注册(P1,迁移 000197;决策 adopted/2026-09-06-odn-business-linkage)。
// 全部 handler 实现见 odn_coverage_handlers.go;此处只保留扁平路由表 + 请求体定义。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// odnCoverageReq 覆盖关联保存请求体(fields.md 1.5.7)。
type odnCoverageReq struct {
	AddressID    int64  `json:"addressId" binding:"required,min=1"`
	FacilityCode string `json:"facilityCode"`
	DeviceID     int64  `json:"deviceId"`
	Status       string `json:"status" binding:"required,oneof=SERVED PENDING UNSERVED"`
	Note         string `json:"note"`
}

// registerODNCoverageRoutes 注册覆盖关联路由(menu:odn 门禁;契约 admin/odn.yaml)。
func registerODNCoverageRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.POST("/odn/coverage", perm, odnUpsertCoverageHandler(a))
	g.GET("/odn/coverage", perm, odnGetCoverageHandler(a))
	g.GET("/odn/coverage/list", perm, odnListCoverageHandler(a))
	g.GET("/odn/coverage/resolve", perm, odnResolveCoverageHandler(a))
}
