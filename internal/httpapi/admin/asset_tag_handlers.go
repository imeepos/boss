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
