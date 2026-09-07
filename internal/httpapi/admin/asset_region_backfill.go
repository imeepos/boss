package adminapi

// 资产区域快照回补(W3 收口,存量开户导入遗留①):POST /assets/region-backfill。
// 幂等可重跑(仅回补空快照行);写操作入审计;失败由域层留 [asset-region-backfill] 日志。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// assetRegionBackfillHandler POST /assets/region-backfill:按地址行级节点归属回补
// region_id/region_name 空快照;返回回补数/未回补数/样本(供验收抽样断言)。
func assetRegionBackfillHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := a.Asset.BackfillRegionSnapshots(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "asset.region-backfill", "assets", "bulk",
			map[string]any{"backfilled": res.Backfilled, "skipped": res.Skipped})
		respond(c, apitypes.CodeOK, gin.H{"result": res})
	}
}
