package app

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerResourceRoutes 注册网络资源域路由(承接 api/openapi/admin/oss.yaml)。
func registerResourceRoutes(g *gin.RouterGroup, a *Application) {
	res := g.Group("", requirePerm(a.User, "menu:resource"))
	res.GET("/resources", func(c *gin.Context) {
		list, err := a.Resource.ListResources(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	res.GET("/ports", func(c *gin.Context) {
		list, err := a.Resource.ListPorts(c.Request.Context(), queryInt64(c, "resourceId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	res.GET("/ports/:portId/change-history", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("portId"), 10, 64)
		list, err := a.ResourceSub.ListPortHistory(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	res.GET("/reserves", func(c *gin.Context) {
		list, err := a.ResourceSub.ListReserveRecords(c.Request.Context(), queryInt64(c, "portId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	tr := g.Group("", requirePerm(a.User, "menu:transfer"))
	tr.GET("/transfers", func(c *gin.Context) {
		list, err := a.ResourceSub.ListTransfers(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	tr.GET("/expansions", func(c *gin.Context) {
		list, err := a.ResourceSub.ListExpansions(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	tr.GET("/expansions/qos-templates", func(c *gin.Context) {
		list, err := a.ResourceAssign.ListQosTemplates(c.Request.Context(), queryInt64(c, "legalEntityId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
}
