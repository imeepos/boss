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
		q, err := a.QuadLink.GetByAsset(c.Request.Context(), queryInt64(c, "assetId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, q)
	})

	g.GET("/quad-links/by-customer", requirePerm(a.User, "menu:quadlink"), func(c *gin.Context) {
		q, err := a.QuadLink.GetByCustomer(c.Request.Context(), queryInt64(c, "customerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, q)
	})

	g.GET("/quad-links/by-port", requirePerm(a.User, "menu:quadlink"), func(c *gin.Context) {
		q, err := a.QuadLink.GetByPort(c.Request.Context(), queryInt64(c, "portId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, q)
	})

	g.GET("/quad-links/by-address", requirePerm(a.User, "menu:quadlink"), func(c *gin.Context) {
		q, err := a.QuadLink.GetByAddress(c.Request.Context(), queryInt64(c, "addressId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, q)
	})
}
