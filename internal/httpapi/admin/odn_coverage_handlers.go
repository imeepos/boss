package adminapi

// ODN 覆盖关联路由 handler 实现(承接 registerODNCoverageRoutes)。
// P1 覆盖关联:可查可判不做下单硬校验(决策 adopted/2026-09-06-odn-business-linkage)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// odnUpsertCoverageHandler POST /odn/coverage:保存地址覆盖关联(一址一覆盖幂等)。
func odnUpsertCoverageHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnCoverageReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		cov := odn.Coverage{AddressID: req.AddressID, FacilityCode: req.FacilityCode,
			DeviceID: req.DeviceID, Status: req.Status, Note: req.Note}
		if err := a.ODN.UpsertCoverage(c.Request.Context(), cov); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"addressId": cov.AddressID, "status": cov.Status})
	}
}

// odnGetCoverageHandler GET /odn/coverage?addressId=:单地址覆盖查询(未登记返回 null)。
func odnGetCoverageHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		addressID, _ := strconv.ParseInt(c.Query("addressId"), 10, 64)
		if addressID <= 0 {
			respond(c, apitypes.CodeInvalidParam, gin.H{"reason": "addressId required"})
			return
		}
		cov, err := a.ODN.GetCoverageByAddress(c.Request.Context(), addressID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, cov)
	}
}

// odnListCoverageHandler GET /odn/coverage/list?limit=:覆盖关联列表(近更优先)。
func odnListCoverageHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		list, err := a.ODN.ListCoverage(c.Request.Context(), limit)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnResolveCoverageHandler GET /odn/coverage/resolve?lat=&lng=:就近设施可装性判定。
func odnResolveCoverageHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		lat, latErr := strconv.ParseFloat(c.Query("lat"), 64)
		lng, lngErr := strconv.ParseFloat(c.Query("lng"), 64)
		if latErr != nil || lngErr != nil {
			respond(c, apitypes.CodeInvalidParam, gin.H{"reason": "lat/lng required"})
			return
		}
		res, err := a.ODN.ResolveLatLng(c.Request.Context(), lat, lng)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, res)
	}
}
