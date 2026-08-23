package adminapi

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func partnerRegionScopeHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		scope, err := a.PartnerRegion.GetRegionScope(c.Request.Context(), claims.AccountID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, scope)
	}
}

func partnerRegionScopeUpdateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RegionPath string `json:"regionPath"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		if err := a.PartnerRegion.SetRegionScope(c.Request.Context(), claims.AccountID, req.RegionPath); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "partner_region_scope.update", "account", strconv.FormatInt(claims.AccountID, 10), gin.H{"regionPath": req.RegionPath})
		respond(c, apitypes.CodeOK, gin.H{"regionPath": req.RegionPath})
	}
}
