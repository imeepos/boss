package adminapi

// 资产域路由注册(承接 api/openapi/admin/asset.yaml)。
// 全部 handler 实现见 asset_handlers.go;此处只保留扁平路由表。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerAssetRoutes 注册资产域路由(承接 api/openapi/admin/asset.yaml)。
func registerAssetRoutes(g *gin.RouterGroup, a *app.Application) {
	ams := g.Group("", requirePerm(a.User, "menu:asset"))
	ams.GET("/assets", assetListHandler(a))
	ams.GET("/assets/:assetId/lifecycle", assetListLifecyclesHandler(a))
	ams.GET("/assets/batches", assetListBatchesHandler(a))
	ams.GET("/assets/assignments", assetListAssignmentsHandler(a))

	g.GET("/tags", requirePerm(a.User, "menu:tag"), tagListHandler(a))

	g.POST("/stocktakes", requirePerm(a.User, "menu:stock"), stocktakeCreateHandler(a))
	g.POST("/stocktakes/:taskId/diff-handle", requirePerm(a.User, "menu:stock"), stocktakeHandleDiffHandler(a))
	g.POST("/replacements", requirePerm(a.User, "menu:replace"), replacementCreateHandler(a))

	g.GET("/stocktakes", requirePerm(a.User, "menu:stock"), stocktakeListHandler(a))
	g.GET("/replacements", requirePerm(a.User, "menu:replace"), replacementListHandler(a))
}