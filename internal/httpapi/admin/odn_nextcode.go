package adminapi

// ODN 下一可用编码只读路由(menu:odn 门禁;契约 admin/odn.yaml)。
// 前端新增表单自动顺延预览,退役占号不复用由域层 MAX+1 保证。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerODNNextCodeRoutes 注册下一可用编码路由。
func registerODNNextCodeRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.GET("/odn/facility-next-code", perm, odnNextFacilityCodeHandler(a))
	g.GET("/odn/site-next-no", perm, odnNextSiteNoHandler(a))
}

// odnNextFacilityCodeHandler GET /odn/facility-next-code?kind=P&gridCode=12。
// P/MH 必带 gridCode;TW/CLS/TBX 市域顺序不带。
func odnNextFacilityCodeHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		kind := c.Query("kind")
		gridCode := 0
		if v, err := strconv.Atoi(c.Query("gridCode")); err == nil {
			gridCode = v
		}
		code, err := a.ODN.NextFacilityCode(c.Request.Context(), kind, int16(gridCode))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"code": code})
	}
}

// odnNextSiteNoHandler GET /odn/site-next-no?prvCode=&cityPrefix=。
func odnNextSiteNoHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		prv, city := c.Query("prvCode"), c.Query("cityPrefix")
		if prv == "" || city == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		no, err := a.ODN.NextSiteNo(c.Request.Context(), prv, city)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"siteNo": no})
	}
}
