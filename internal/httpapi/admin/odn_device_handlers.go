package adminapi

// ODN 局点/核心链路设备路由 handler 实现(承接 registerODNSiteDeviceRoutes)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// odnListSitesHandler GET /odn/sites:局点列表(按省/市过滤)。
func odnListSitesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.ODN.ListSites(c.Request.Context(), c.Query("prvCode"), c.Query("cityPrefix"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnCreateSiteHandler POST /odn/sites:新建局点(fields.md 1.5.5)。
func odnCreateSiteHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnSiteReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		st := odn.Site{PrvCode: c.Query("prvCode"), CityPrefix: c.Query("cityPrefix"),
			SiteNo: req.SiteNo, Name: req.Name, Lat: req.Lat, Lng: req.Lng}
		if st.PrvCode == "" || st.CityPrefix == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := a.ODN.CreateSite(c.Request.Context(), st); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"nodeCode": st.NodeCode()})
	}
}

// odnRetireSiteHandler DELETE /odn/sites/{siteNo}:局点报废。
func odnRetireSiteHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		siteNo, err := strconv.Atoi(c.Param("siteNo"))
		if err != nil || siteNo < 1 || siteNo > 999 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := a.ODN.RetireSite(c.Request.Context(),
			c.Query("prvCode"), c.Query("cityPrefix"), int16(siteNo)); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	}
}

// odnListDevicesHandler GET /odn/devices:设备列表(按 kind/省/市过滤)。
func odnListDevicesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.ODN.ListDevices(c.Request.Context(),
			c.Query("kind"), c.Query("prvCode"), c.Query("cityPrefix"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnCreateDeviceHandler POST /odn/devices:新建设备。
func odnCreateDeviceHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnDeviceReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		d := odn.Device{Code: req.Code, Kind: req.Kind, PrvCode: req.PrvCode,
			CityPrefix: req.CityPrefix, SiteNo: req.SiteNo, ParentID: req.ParentID,
			Name: req.Name, Lat: req.Lat, Lng: req.Lng}
		if err := a.ODN.CreateDevice(c.Request.Context(), d); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"code": d.Code})
	}
}

// odnRetireDeviceHandler DELETE /odn/devices/{id}:设备报废。
func odnRetireDeviceHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.ODN.RetireDevice(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	}
}
