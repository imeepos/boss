package adminapi

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// odnSiteReq 局点新建请求体(fields.md 1.5.5)。
type odnSiteReq struct {
	SiteNo int16   `json:"siteNo" binding:"required,min=1,max=999"`
	Name   string  `json:"name"`
	Lat    float64 `json:"lat"`
	Lng    float64 `json:"lng"`
}

// odnDeviceReq 设备新建请求体。
type odnDeviceReq struct {
	Code       string `json:"code" binding:"required"`
	Kind       string `json:"kind" binding:"required,oneof=SNW OLT ODF OCC ODB SDB PRT TBP"`
	PrvCode    string `json:"prvCode" binding:"required,len=6"`
	CityPrefix string `json:"cityPrefix" binding:"required,min=3,max=5"`
	SiteNo     int16  `json:"siteNo" binding:"min=0,max=999"`
	ParentID   int64  `json:"parentId"`
	Name       string `json:"name"`
}

// registerODNSiteDeviceRoutes 局点与核心链路设备路由。
func registerODNSiteDeviceRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.GET("/odn/sites", perm, func(c *gin.Context) {
		list, err := a.ODN.ListSites(c.Request.Context(), c.Query("prvCode"), c.Query("cityPrefix"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})
	g.POST("/odn/sites", perm, func(c *gin.Context) {
		var req odnSiteReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
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
	})
	g.DELETE("/odn/sites/:siteNo", perm, func(c *gin.Context) {
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
	})

	g.GET("/odn/devices", perm, func(c *gin.Context) {
		list, err := a.ODN.ListDevices(c.Request.Context(),
			c.Query("kind"), c.Query("prvCode"), c.Query("cityPrefix"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})
	g.POST("/odn/devices", perm, func(c *gin.Context) {
		var req odnDeviceReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		d := odn.Device{Code: req.Code, Kind: req.Kind, PrvCode: req.PrvCode,
			CityPrefix: req.CityPrefix, SiteNo: req.SiteNo, ParentID: req.ParentID, Name: req.Name}
		if err := a.ODN.CreateDevice(c.Request.Context(), d); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"code": d.Code})
	})
	g.DELETE("/odn/devices/:id", perm, func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := a.ODN.RetireDevice(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	})
}
