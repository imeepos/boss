package adminapi

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerODNFacilityRoutes 设施路由(列表/详情/新建/报废)。
func registerODNFacilityRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.GET("/odn/facilities", perm, func(c *gin.Context) {
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
	})
	g.GET("/odn/facilities/:code", perm, func(c *gin.Context) {
		f, err := a.ODN.GetFacility(c.Request.Context(), c.Param("code"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, f)
	})
	g.POST("/odn/facilities", perm, func(c *gin.Context) {
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
	})
	g.DELETE("/odn/facilities/:code", perm, func(c *gin.Context) {
		if err := a.ODN.RetireFacility(c.Request.Context(), c.Param("code")); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	})
}

// registerODNCableRoutes 光缆段落/纤芯路由。
func registerODNCableRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.GET("/odn/segments", perm, func(c *gin.Context) {
		list, err := a.ODN.ListSegments(c.Request.Context(), c.Query("endpoint"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})
	g.POST("/odn/segments", perm, func(c *gin.Context) {
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
	})
	g.POST("/odn/segments/:id/fibers", perm, func(c *gin.Context) {
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
	})
	g.GET("/odn/segments/:id/fibers", perm, func(c *gin.Context) {
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
	})
}
