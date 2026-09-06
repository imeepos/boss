package adminapi

// 标签域管理端 handler(P2-W2-T1):建标签/禁用/启用/事件流。
// 路由挂 registerAssetRoutes(menu:tag,与既有 /tags 列表/解绑同权限)。

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// tagCreateHandler POST /tags:建标签。编号+EPC+频段必填且唯一
// (唯一冲突 40900,ErrCodeDuplicate);状态缺省 UNBOUND;法人必填
// (tags.legal_entity_id NOT NULL FK,服务层校验存在性)。
func tagCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var t asset.Tag
		if !httpx.BindAndValidate(c, &t, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(t.LegalEntityID, "legalEntityId"),
				httpx.RequireString(t.TagNo, "tagNo", 32),
				httpx.RequireString(t.EpcCode, "epcCode", 32),
				httpx.RequireString(t.Band, "band", 8),
			)
		}) {
			return
		}
		if t.Status == "" {
			t.Status = "UNBOUND"
		}
		id, err := a.Asset.CreateTag(c.Request.Context(), t)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "tag", fmt.Sprint(id), map[string]any{
			"op": "create", "tagNo": t.TagNo, "epcCode": t.EpcCode, "band": t.Band,
		})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// tagDisableHandler POST /tags/{tagId}/disable:停用标签(P2-W2-T1 B)。
// 仅 UNBOUND 可停用(BOUND 40900 提示先解绑);已 DISABLED 幂等成功。
func tagDisableHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "tagId")
		if !ok {
			return
		}
		var req struct {
			Reason string `json:"reason"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if err := a.Asset.DisableTag(c.Request.Context(), id, req.Reason); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "tag", c.Param("tagId"), map[string]any{"op": "disable", "reason": req.Reason})
		respond(c, apitypes.CodeOK, nil)
	}
}

// tagEnableHandler POST /tags/{tagId}/enable:启用标签(P2-W2-T1 B)。
// 仅对 DISABLED 生效(回 UNBOUND);非 DISABLED 幂等 no-op 成功。
func tagEnableHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "tagId")
		if !ok {
			return
		}
		if err := a.Asset.EnableTag(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "tag", c.Param("tagId"), map[string]any{"op": "enable"})
		respond(c, apitypes.CodeOK, nil)
	}
}

// tagEventsHandler GET /tags/{tagId}/events:标签事件流(P2-W2-T1 C)。
// append-only 审计流只读,BIND/UNBIND/RECYCLE 按时间倒序回放。
func tagEventsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "tagId")
		if !ok {
			return
		}
		list, err := a.Asset.ListTagEvents(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}
