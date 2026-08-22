package adminapi

// 经营区域 handler 实现(org_region.go 仅留路由表)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func regionList(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.User.ListRegions(c.Request.Context(), c.Query("parentPath"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

func regionAssignCoverage(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}
