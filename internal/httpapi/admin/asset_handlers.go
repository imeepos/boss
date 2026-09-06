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

// assetListHandler GET /assets:资产主档分页列表(P3-T1)。
// offset/limit/status/type/modelId/q 过滤+排序白名单;sort 越界 parse 层 400。
func assetListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		q, ok := parseListQuery(c, asset.AssetSortColumns)
		if !ok {
			return
		}
		page, err := a.Asset.ListAssetsPage(c.Request.Context(), q)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": page.Items, "total": page.Total})
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

// tagListHandler GET /tags:标签字典分页列表(P3-T1)。
// offset/limit/status/q(tag_no 前缀)过滤+排序白名单;sort 越界 parse 层 400。
func tagListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		q, ok := parseListQuery(c, asset.TagSortColumns)
		if !ok {
			return
		}
		page, err := a.Asset.ListTagsPage(c.Request.Context(), q)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": page.Items, "total": page.Total})
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

// replacementAssignHandler POST /replacements/{id}/assign:派单(指派师傅,PENDING→DOING)。
func replacementAssignHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var req struct {
			WorkerID int64 `json:"workerId"`
		}
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(httpx.RequirePositiveID(req.WorkerID, "workerId"))
		}) {
			return
		}
		w, err := a.Worker.GetWorker(c.Request.Context(), req.WorkerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !workerAssignable(w) {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "worker not active"})
			return
		}
		r, err := a.Asset.AssignReplacement(c.Request.Context(), id, w.ID, w.Name)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "replacement", r.ReplacementNo,
			map[string]any{"workerId": r.WorkerID, "workerName": r.WorkerName})
		respond(c, apitypes.CodeOK, r)
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

// assetScrapHandler POST /assets/{assetId}/scrap:报废资产(P1-T2)。
// 终态幂等;标签仍绑时强制解绑写 RECYCLE 事件,轨迹落 SCRAP 行(同事务)。
func assetScrapHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "assetId")
		if !ok {
			return
		}
		var req struct {
			Reason string `json:"reason"`
		}
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.Reason, "reason", 64),
			)
		}) {
			return
		}
		if err := a.Asset.ScrapAsset(c.Request.Context(), id, httpx.ClaimsAccountID(c), req.Reason); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "asset", c.Param("assetId"), map[string]any{"op": "scrap", "reason": req.Reason})
		respond(c, apitypes.CodeOK, nil)
	}
}

// tagUnbindHandler POST /tags/{tagId}/unbind:解绑标签(P1-T2)。
// expectedAssetId>0 时校验当前绑定一致;成功后标签回 UNBOUND 可复用,写 UNBIND 事件。
func tagUnbindHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "tagId")
		if !ok {
			return
		}
		var req struct {
			ExpectedAssetID int64  `json:"expectedAssetId"`
			Reason          string `json:"reason"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if err := a.Asset.UnbindTag(c.Request.Context(), id, req.ExpectedAssetID, httpx.ClaimsAccountID(c), req.Reason); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "tag", c.Param("tagId"), map[string]any{"op": "unbind", "expectedAssetId": req.ExpectedAssetID})
		respond(c, apitypes.CodeOK, nil)
	}
}

// modelListHandler GET /asset-models:型号字典(P1-T3,含停用,下拉与列表)。
func modelListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Asset.ListModels(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// modelCreateHandler POST /asset-models:建型号(四元组唯一,冲突 40900)。
func modelCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var m asset.AssetModel
		if !httpx.BindAndValidate(c, &m, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(m.Model, "model", 128),
				httpx.RequireString(m.Category, "category", 32),
			)
		}) {
			return
		}
		id, err := a.Asset.CreateModel(c.Request.Context(), m)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "asset_model", fmt.Sprint(id), map[string]any{"model": m.Model, "category": m.Category})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}
