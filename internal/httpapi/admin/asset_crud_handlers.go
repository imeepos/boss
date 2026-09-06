package adminapi

// 资产台账 admin CRUD handler(P2-W1-T1):建档/详情/受限编辑/守卫删除。
// 路由挂 registerAssetRoutes(menu:asset);编辑仅 类型/型号/标签/批次 四键,
// 删除为守卫式物理删,状态与部署地址一律走业务流转不经编辑通道。

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// assetCreateHandler POST /assets:建档(批次必填,企业快照自批次回填,状态固定
// IN_STOCK;型号可选须存在且在用,类型缺省由型号类别派生;标签可选绑定写 BIND
// 事件,已被其他资产占用 40900 整单回滚;assetCode 缺省服务端生成;建档同事务
// 落初始轨迹行)。
func assetCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			AssetCode string `json:"assetCode"`
			BatchID   int64  `json:"batchId"`
			ModelID   int64  `json:"modelId"`
			TagID     int64  `json:"tagId"`
			Type      string `json:"type"`
			SN        string `json:"sn"`
			MAC       string `json:"mac"`
			LOID      string `json:"loid"`
		}
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(req.BatchID, "batchId"),
			)
		}) {
			return
		}
		// 身份三要素归一(P3-T2):SN/LOID 去空格,MAC 校验格式;空串存 NULL,
		// 唯一冲突由 DB 部分唯一索引兜底(ErrAssetIdentityDuplicate,40900)。
		sn, mac, loid, idErr := asset.NormalizeIdentity(req.SN, req.MAC, req.LOID)
		if idErr != nil {
			respondErr(c, idErr)
			return
		}
		id, err := a.Asset.CreateAsset(c.Request.Context(), asset.Asset{
			AssetCode: req.AssetCode,
			BatchID:   req.BatchID,
			ModelID:   req.ModelID,
			TagID:     req.TagID,
			Type:      req.Type,
			Status:    "IN_STOCK",
			SN:        sn,
			MAC:       mac,
			LOID:      loid,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "asset", fmt.Sprint(id), map[string]any{
			"op": "create", "batchId": req.BatchID, "tagId": req.TagID, "modelId": req.ModelID,
			"sn": sn, "mac": mac, "loid": loid,
		})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// assetGetHandler GET /assets/{assetId}:单条主档详情(含企业/区域快照);未命中 40400。
func assetGetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "assetId")
		if !ok {
			return
		}
		row, err := a.Asset.GetAsset(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, row)
	}
}

// assetUpdateHandler PUT /assets/{assetId}:受限编辑(仅 类型/型号/标签/批次;
// 标签换绑同事务 UNBIND+BIND 冲突 40900 整单回滚;批次仅 IN_STOCK 可改并同步
// 企业快照;四键无变化幂等成功;审计记变更前后键值)。
func assetUpdateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "assetId")
		if !ok {
			return
		}
		var in asset.AssetUpdate
		if !httpx.BindAndValidate(c, &in, func() error {
			// 四键均为 0=不改(值类型缺省即未传);仅拒负数,缺省 batchId 的
			// type-only 编辑是合法请求(RequirePositiveID 曾误拒之,42200)。
			return httpx.CollectErrors(
				httpx.RequireNonNegativeID(in.BatchID, "batchId"),
				httpx.RequireNonNegativeID(in.ModelID, "modelId"),
				httpx.RequireNonNegativeID(in.TagID, "tagId"),
			)
		}) {
			return
		}
		before, err := a.Asset.GetAsset(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Asset.UpdateAsset(c.Request.Context(), id, in, httpx.ClaimsAccountID(c)); err != nil {
			respondErr(c, err)
			return
		}
		after, err := a.Asset.GetAsset(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "asset", c.Param("assetId"), updateDiff(before, after))
		respond(c, apitypes.CodeOK, after)
	}
}

// updateDiff 变更前后键值对照(仅列发生变化键;空 map=无变化幂等,审计仍留操作痕)。
func updateDiff(before, after *asset.Asset) map[string]any {
	type getter func(*asset.Asset) any
	gets := []getter{
		func(x *asset.Asset) any { return x.Type },
		func(x *asset.Asset) any { return x.ModelID },
		func(x *asset.Asset) any { return x.TagID },
		func(x *asset.Asset) any { return x.BatchID },
		func(x *asset.Asset) any { return x.LegalEntityID },
		func(x *asset.Asset) any { return x.SN },
		func(x *asset.Asset) any { return x.MAC },
		func(x *asset.Asset) any { return x.LOID },
	}
	names := []string{"type", "modelId", "tagId", "batchId", "legalEntityId", "sn", "mac", "loid"}
	diff := map[string]any{}
	for i, get := range gets {
		if get(before) != get(after) {
			diff[names[i]] = map[string]any{"before": get(before), "after": get(after)}
		}
	}
	return diff
}

// assetDeleteHandler DELETE /assets/{assetId}:守卫删除(仅 IN_STOCK 且无标签绑定/
// 持有台账/换新单/盘点明细/四码关联可物理删;命中任一引用 40900 且 message 列全
// 阻断项;SCRAPPED 一律拒绝硬删提示走报废端点;审计附资产编码载荷)。
func assetDeleteHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "assetId")
		if !ok {
			return
		}
		code, err := a.Asset.DeleteAsset(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "asset", c.Param("assetId"), map[string]any{"op": "delete", "assetCode": code})
		respond(c, apitypes.CodeOK, nil)
	}
}
