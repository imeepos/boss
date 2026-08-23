package adminapi

// ODN 设施/光缆路由 handler 实现(承接 registerODNFacilityRoutes / registerODNCableRoutes)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// odnListFacilitiesHandler GET /odn/facilities:设施列表。
func odnListFacilitiesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		kind := c.Query("kind")
		ref := odn.GridRef{}
		if c.Query("gridCode") != "" {
			ref = odn.GridRef{PrvCode: c.Query("prvCode"), CityPrefix: c.Query("cityPrefix")}
			ref.GridCode = func() int16 {
				v, _ := strconv.Atoi(c.Query("gridCode"))
				return int16(v)
			}()
		}
		list, err := a.ODN.ListFacilities(c.Request.Context(), kind, ref)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnGetFacilityHandler GET /odn/facilities/{code}:设施详情。
func odnGetFacilityHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		f, err := a.ODN.GetFacility(c.Request.Context(), c.Param("code"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, f)
	}
}

// odnCreateFacilityHandler POST /odn/facilities:新建设施。
func odnCreateFacilityHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnFacilityReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		f := odn.Facility{Code: req.Code, Kind: req.Kind, PrvCode: req.PrvCode,
			CityPrefix: req.CityPrefix, GridCode: req.GridCode, Name: req.Name,
			Lat: req.Lat, Lng: req.Lng}
		if err := a.ODN.CreateFacility(c.Request.Context(), f); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"code": f.Code})
	}
}

// odnRetireFacilityHandler DELETE /odn/facilities/{code}:设施报废。
func odnRetireFacilityHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := a.ODN.RetireFacility(c.Request.Context(), c.Param("code")); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	}
}

// odnListSegmentsHandler GET /odn/segments:光缆段落列表。
func odnListSegmentsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.ODN.ListSegments(c.Request.Context(), c.Query("endpoint"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnCreateSegmentHandler POST /odn/segments:新建光缆段落。
func odnCreateSegmentHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnSegmentReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		seg, err := a.ODN.CreateSegment(c.Request.Context(), req.Endpoint1, req.Endpoint2, req.Name)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, seg)
	}
}

// odnAddFiberHandler POST /odn/segments/{id}/fibers:新增纤芯。
func odnAddFiberHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var req odnFiberReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if err := a.ODN.AddFiber(c.Request.Context(), id, odn.Fiber{GNo: req.GNo, Kind: req.Kind}); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	}
}

// odnListFibersHandler GET /odn/segments/{id}/fibers:纤芯列表。
func odnListFibersHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		list, err := a.ODN.ListFibers(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}
