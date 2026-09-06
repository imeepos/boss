package adminapi

// 入库批次管理端 handler(P2-W2-T1 F):建批次。
// 批次列表(P2-W1-T1)在 asset_handlers.go;本文件只收建批次写侧。

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// genBatchCode 批次编码缺省生成:RK-YYYYMMDD-NNNNN,对齐采购入库 nextBatchCode
// 风格(procurement/pg_write.go,17 字符恒满足 asset_batches.code VARCHAR(32));
// 序号取纳秒尾数防同日撞号,DB code UNIQUE 兜底。
func genBatchCode() string {
	now := time.Now().UTC()
	return fmt.Sprintf("RK-%s-%05d", now.Format("20060102"), now.UnixNano()%100000)
}

// batchCreateHandler POST /asset-batches:建批次(P2-W2-T1 F)。
// 名称+法人主体必填;批次编码缺省自动生成(RK 风格);法人不存在服务层
// 校验错(ErrForeignKeyViolation → 42200)。
func batchCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var b asset.AssetBatch
		if !httpx.BindAndValidate(c, &b, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(b.LegalEntityID, "legalEntityId"),
				httpx.RequireString(b.Name, "name", 64),
			)
		}) {
			return
		}
		if b.Code == "" {
			b.Code = genBatchCode()
		}
		id, err := a.Asset.CreateBatch(c.Request.Context(), b)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "asset_batch", fmt.Sprint(id), map[string]any{
			"op": "create", "code": b.Code, "name": b.Name, "legalEntityId": b.LegalEntityID,
		})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "code": b.Code})
	}
}
