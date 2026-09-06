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
	ams.POST("/assets/:assetId/scrap", assetScrapHandler(a))  // P1-T2 报废(标签强回收+事件流)
	ams.GET("/assets/:assetId/events", assetEventsHandler(a)) // P2-T4 事件消费面(组级 menu:asset)
	ams.GET("/asset-models", modelListHandler(a))             // P1-T3 型号字典
	ams.POST("/asset-models", modelCreateHandler(a))

	g.GET("/tags", requirePerm(a.User, "menu:tag"), tagListHandler(a))
	g.GET("/tags/:tagId/events", requirePerm(a.User, "menu:tag"), tagEventsHandler(a))  // P2-T4 事件消费面
	g.POST("/tags/:tagId/unbind", requirePerm(a.User, "menu:tag"), tagUnbindHandler(a)) // P1-T2 解绑回收

	g.POST("/stocktakes", requirePerm(a.User, "menu:stock"), stocktakeCreateHandler(a))
	g.POST("/stocktakes/:taskId/diff-handle", requirePerm(a.User, "menu:stock"), stocktakeHandleDiffHandler(a))
	g.POST("/stocktakes/:taskId/scans", requirePerm(a.User, "menu:stock"), stocktakeScanHandler(a))
	g.GET("/stocktakes/:taskId/items", requirePerm(a.User, "menu:stock"), stocktakeItemsHandler(a))
	g.POST("/stocktakes/:taskId/items/:itemId/handle", requirePerm(a.User, "menu:stock"), stocktakeItemHandleHandler(a))
	g.POST("/replacements", requirePerm(a.User, "menu:replace"), replacementCreateHandler(a))
	g.POST("/replacements/:id/assign", requirePerm(a.User, "menu:replace"), replacementAssignHandler(a))

	g.GET("/stocktakes", requirePerm(a.User, "menu:stock"), stocktakeListHandler(a))
	g.GET("/replacements", requirePerm(a.User, "menu:replace"), replacementListHandler(a))
}
