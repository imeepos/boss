package app

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerQuadlinkRoutes 注册四码合一域路由(承接 quad.yaml,任一码反查)。
func registerQuadlinkRoutes(g *gin.RouterGroup, a *Application) {
	g.GET("/quad-links", requirePerm(a.User, "menu:quadlink"), func(c *gin.Context) {
		list, err := a.QuadLink.ListLinks(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/quad-links/by-asset", requirePerm(a.User, "menu:quadlink"), func(c *gin.Context) {
		assetID, ok := requireID(c, "assetId")
		if !ok {
			return
		}
		q, err := a.QuadLink.GetByAsset(c.Request.Context(), assetID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, q)
	})

	g.GET("/quad-links/by-customer", requirePerm(a.User, "menu:quadlink"), func(c *gin.Context) {
		customerID, ok := requireID(c, "customerId")
		if !ok {
			return
		}
		q, err := a.QuadLink.GetByCustomer(c.Request.Context(), customerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, q)
	})

	g.GET("/quad-links/by-port", requirePerm(a.User, "menu:quadlink"), func(c *gin.Context) {
		portID, ok := requireID(c, "portId")
		if !ok {
			return
		}
		q, err := a.QuadLink.GetByPort(c.Request.Context(), portID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, q)
	})

	g.GET("/quad-links/by-address", requirePerm(a.User, "menu:quadlink"), func(c *gin.Context) {
		addressID, ok := requireID(c, "addressId")
		if !ok {
			return
		}
		q, err := a.QuadLink.GetByAddress(c.Request.Context(), addressID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, q)
	})
}

// requireID 校验必填 ID 查询参数:缺失/非法/非正数时返回 422 并终止请求。
func requireID(c *gin.Context, name string) (int64, bool) {
	v := queryInt64(c, name)
	if v <= 0 {
		respond(c, apitypes.CodeInvalidParam, nil)
		return 0, false
	}
	return v, true
}