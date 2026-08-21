package adminapi

// 经营区域路由(list + 覆盖主体划分,从 org.go 拆出,保持单文件 ≤300 行)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerRegionRoutes 经营区域路由(g 组)。
func registerRegionRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/regions", requirePerm(a.User, "menu:region"), func(c *gin.Context) {
		list, err := a.User.ListRegions(c.Request.Context(), c.Query("parentPath"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	// 区域覆盖主体划分:子公司划经营区域/摘除(migrations/000076;0=摘除回落祖先/总公司兜底)。
	g.PUT("/regions/:regionId/coverage", requirePerm(a.User, "menu:region"), func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "regionId")
		if !ok {
			return
		}
		var req struct {
			LegalEntityID int64 `json:"legalEntityId"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if err := a.User.AssignRegionCoverage(c.Request.Context(), id, req.LegalEntityID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "org.assign-region-coverage", "region", c.Param("regionId"),
			map[string]any{"legalEntityId": req.LegalEntityID})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

}
