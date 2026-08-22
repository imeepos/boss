package adminapi

// ODN 局点与核心链路设备路由注册。
// 全部 handler 实现见 odn_device_handlers.go;此处只保留扁平路由表 + 请求体定义。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
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

// registerODNSiteDeviceRoutes 注册局点与核心链路设备路由。
func registerODNSiteDeviceRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.GET("/odn/sites", perm, odnListSitesHandler(a))
	g.POST("/odn/sites", perm, odnCreateSiteHandler(a))
	g.DELETE("/odn/sites/:siteNo", perm, odnRetireSiteHandler(a))

	g.GET("/odn/devices", perm, odnListDevicesHandler(a))
	g.POST("/odn/devices", perm, odnCreateDeviceHandler(a))
	g.DELETE("/odn/devices/:id", perm, odnRetireDeviceHandler(a))
}