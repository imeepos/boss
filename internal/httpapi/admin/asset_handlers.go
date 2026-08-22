package adminapi

// 资产域路由具名 handler(承接 registerAssetRoutes 扁平路由表)。
// 主档 / 生命周期 / 批次 / 指派记录 / 标签 / 盘点 / 替换。

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// assetListHandler GET /assets:资产主档列表。
func assetListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Asset.ListAssets(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// assetListLifecyclesHandler GET /assets/{assetId}/lifecycle:资产生命周期记录。
func assetListLifecyclesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "assetId")
		if !ok {
			return
		}
		list, err := a.Asset.ListLifecycles(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// assetListBatchesHandler GET /assets/batches:资产批次列表。
func assetListBatchesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Asset.ListBatches(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// assetListAssignmentsHandler GET /assets/assignments:资产指派记录(按 assetId 过滤)。
func assetListAssignmentsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Asset.ListAssignments(c.Request.Context(), queryInt64(c, "assetId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// tagListHandler GET /tags:标签字典列表。
func tagListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Asset.ListTags(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// stocktakeCreateHandler POST /stocktakes:新建盘点任务。
func stocktakeCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var st asset.Stocktake
		if !httpx.BindAndValidate(c, &st, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(st.LegalEntityID, "legalEntityId"),
				httpx.RequireString(st.Scope, "scope", 64),
			)
		}) {
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
		httpx.RecordAudit(a, c, "数据变更", "stocktake", fmt.Sprint(id), map[string]any{"scope": st.Scope})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// stocktakeHandleDiffHandler POST /stocktakes/{taskId}/diff-handle:处理盘点差异。
func stocktakeHandleDiffHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "taskId")
		if !ok {
			return
		}
		if err := a.Asset.HandleStocktakeDiff(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "stocktake", c.Param("taskId"), nil)
		respond(c, apitypes.CodeOK, nil)
	}
}

// replacementCreateHandler POST /replacements:新建替换单。
func replacementCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r asset.Replacement
		if !httpx.BindAndValidate(c, &r, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(r.AssetID, "assetId"),
			)
		}) {
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
		httpx.RecordAudit(a, c, "数据变更", "replacement", r.ReplacementNo, map[string]any{"assetId": r.AssetID})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "replacementNo": r.ReplacementNo})
	}
}

// stocktakeListHandler GET /stocktakes:盘点任务列表。
func stocktakeListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Asset.ListStocktakes(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// replacementListHandler GET /replacements:替换单列表。
func replacementListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Asset.ListReplacements(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}