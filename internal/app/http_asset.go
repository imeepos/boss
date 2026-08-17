package app

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerAssetRoutes 注册资产域路由(承接 api/openapi/admin/asset.yaml)。
func registerAssetRoutes(g *gin.RouterGroup, a *Application) {
	ams := g.Group("", requirePerm(a.User, "menu:asset"))
	ams.GET("/assets", func(c *gin.Context) {
		list, err := a.Asset.ListAssets(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	ams.GET("/assets/:assetId/lifecycle", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("assetId"), 10, 64)
		list, err := a.Asset.ListLifecycles(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	ams.GET("/assets/batches", func(c *gin.Context) {
		list, err := a.Asset.ListBatches(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	ams.GET("/assets/assignments", func(c *gin.Context) {
		list, err := a.Asset.ListAssignments(c.Request.Context(), queryInt64(c, "assetId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/tags", requirePerm(a.User, "menu:tag"), func(c *gin.Context) {
		list, err := a.Asset.ListTags(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/stocktakes", requirePerm(a.User, "menu:stock"), func(c *gin.Context) {
		list, err := a.Asset.ListStocktakes(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/replacements", requirePerm(a.User, "menu:replace"), func(c *gin.Context) {
		list, err := a.Asset.ListReplacements(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
}
