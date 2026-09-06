package adminapi

// 标签/资产事件流查询 handler(P2-T4 消费面,承接 api/openapi/admin/asset.yaml)。
// 口径:id 倒序 + limit(默认 50 上限 100,服务层再兜底)+ action 多值白名单过滤 +
// before_id 游标(翻页预留);统一信封,ErrNotFound → 40400。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// eventListLimit 事件流 limit:默认 50,上限 100(服务层兜底再夹一次)。
func eventListLimit(c *gin.Context) int64 {
	l := queryInt64(c, "limit")
	if l <= 0 {
		return 50
	}
	if l > 100 {
		return 100
	}
	return l
}

// eventListActions 事件流 action 过滤:多值白名单,越界值 42200 拒收(防误配静默空结果)。
func eventListActions(c *gin.Context) ([]string, bool) {
	raw := c.QueryArray("action")
	for _, a := range raw {
		if !asset.ValidEventAction(a) {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "invalid action: " + a})
			return nil, false
		}
	}
	return raw, true
}

// tagEventsHandler GET /tags/{tagId}/events:标签事件流(P2-T4,倒序+limit+白名单过滤)。
func tagEventsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "tagId")
		if !ok {
			return
		}
		actions, ok := eventListActions(c)
		if !ok {
			return
		}
		items, err := a.Asset.ListTagEvents(c.Request.Context(), id, eventListLimit(c), queryInt64(c, "before_id"), actions)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// assetEventsHandler GET /assets/{assetId}/events:资产事件流(P2-T4,口径同标签侧)。
func assetEventsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "assetId")
		if !ok {
			return
		}
		actions, ok := eventListActions(c)
		if !ok {
			return
		}
		items, err := a.Asset.ListAssetEvents(c.Request.Context(), id, eventListLimit(c), queryInt64(c, "before_id"), actions)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}
