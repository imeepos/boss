package adminapi

// ODN 省市编码字典只读路由(menu:odn 门禁;契约 admin/odn.yaml)。
// 数据源 000075 种子表,只读;前端新增表单省市级联下拉数据源。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerODNDictRoutes 注册省市字典路由。
func registerODNDictRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.GET("/odn/regions", perm, odnListRegionsHandler(a))
	g.GET("/odn/cities", perm, odnListCitiesHandler(a))
}

// odnListRegionsHandler GET /odn/regions:省级编码字典。
func odnListRegionsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.ODN.ListRegions(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnListCitiesHandler GET /odn/cities?prvCode=:省内城市前缀字典。
func odnListCitiesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		prv := c.Query("prvCode")
		if prv == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		list, err := a.ODN.ListCities(c.Request.Context(), prv)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}
