package adminapi

// 型号字典管理端 handler(P2-W2-T1 D/E):编辑/停用/启用。
// 列表与建型号(P1-T3)在 asset_handlers.go;本文件只收管理写侧扩展。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// modelUpdateHandler PUT /asset-models/{id}:编辑型号(P2-W2-T1 D)。
// 厂商/型号名/类别/料号/规格可改;唯一冲突 40900;停用型号 40900 先启用。
func modelUpdateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var m asset.AssetModel
		if !httpx.BindAndValidate(c, &m, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(m.Model, "model", 128),
				httpx.RequireString(m.Category, "category", 32),
			)
		}) {
			return
		}
		if err := a.Asset.UpdateModel(c.Request.Context(), id, m); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "asset_model", c.Param("id"), map[string]any{
			"op": "update", "vendor": m.Vendor, "model": m.Model, "category": m.Category, "partNumber": m.PartNumber,
		})
		respond(c, apitypes.CodeOK, nil)
	}
}

// modelDisableHandler POST /asset-models/{id}/disable:停用型号(P2-W2-T1 E)。
// is_active 置否,不物理删;已被资产引用由 model_id 承载;幂等。
func modelDisableHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.Asset.SetModelActive(c.Request.Context(), id, false); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "asset_model", c.Param("id"), map[string]any{"op": "disable"})
		respond(c, apitypes.CodeOK, nil)
	}
}

// modelEnableHandler POST /asset-models/{id}/enable:启用型号(P2-W2-T1 E)。
// is_active 置真;幂等(已启用重复启用成功)。
func modelEnableHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.Asset.SetModelActive(c.Request.Context(), id, true); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "asset_model", c.Param("id"), map[string]any{"op": "enable"})
		respond(c, apitypes.CodeOK, nil)
	}
}
