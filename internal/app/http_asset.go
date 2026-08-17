package app

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/asset"
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

	g.POST("/stocktakes", requirePerm(a.User, "menu:stock"), func(c *gin.Context) {
		var st asset.Stocktake
		if err := c.ShouldBindJSON(&st); err != nil || st.LegalEntityID == 0 || st.Scope == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if st.Status == "" {
			st.Status = "DOING"
		}
		id, err := a.Asset.CreateStocktake(c.Request.Context(), st)
		if err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "数据变更", "stocktake", fmt.Sprint(id), map[string]any{"scope": st.Scope})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})
	g.POST("/stocktakes/:taskId/diff-handle", requirePerm(a.User, "menu:stock"), func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("taskId"), 10, 64)
		if err := a.Asset.HandleStocktakeDiff(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "状态变更", "stocktake", c.Param("taskId"), nil)
		respond(c, apitypes.CodeOK, nil)
	})
	g.POST("/replacements", requirePerm(a.User, "menu:replace"), func(c *gin.Context) {
		var r asset.Replacement
		if err := c.ShouldBindJSON(&r); err != nil || r.AssetID == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if r.ReplacementNo == "" {
			r.ReplacementNo = genNo("RPL")
		}
		if r.Status == "" {
			r.Status = "PENDING"
		}
		id, err := a.Asset.CreateReplacement(c.Request.Context(), r)
		if err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "数据变更", "replacement", r.ReplacementNo, map[string]any{"assetId": r.AssetID})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "replacementNo": r.ReplacementNo})
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
