package adminapi

// ODN 影响面查询路由(P7,T10;运维侧"一缆断/一设施坏影响谁")。
// 只读聚合:coverage 挂接 → 地址 → 客户清单;规则见 internal/domain/odn/pg_impact.go。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerODNImpactRoutes 注册影响面查询路由(menu:odn 门禁;契约 admin/odn.yaml)。
func registerODNImpactRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.GET("/odn/impact", perm, odnImpactHandler(a))
}

// odnImpactHandler GET /odn/impact?facilityCode=:设施维度影响面报告。
func odnImpactHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		facilityCode := c.Query("facilityCode")
		if facilityCode == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"reason": "facilityCode required"})
			return
		}
		rep, err := a.ODN.ImpactByFacility(c.Request.Context(), facilityCode)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.impact.query", "odn_facility", facilityCode, nil)
		respond(c, apitypes.CodeOK, rep)
	}
}
