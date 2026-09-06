// 资产/标签列表分页 handler(P3-T1):参数解析见 asset_page_query.go,查询见 domain asset pg_page.go。
package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
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
