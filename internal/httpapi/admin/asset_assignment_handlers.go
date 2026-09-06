package adminapi

// 资产持有台账管理端 handler(P2-W2-T1 G/H):领用/归还。
// 台账列表(P2-W1-T1)在 asset_handlers.go;本文件只收领用与归还写侧。

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// assignmentCreateHandler POST /asset-assignments:领用(P2-W2-T1 G)。
// 资产+师傅+事由必填;仅 IN_STOCK 可领用(40900);落持有台账开段。
// 资产状态不改:装机扫码(000185)才置 DEPLOYED,领用只落台账事实。
func assignmentCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var asg asset.AssetAssignment
		if !httpx.BindAndValidate(c, &asg, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(asg.AssetID, "assetId"),
				httpx.RequirePositiveID(asg.WorkerID, "workerId"),
				httpx.RequireString(asg.Reason, "reason", 128),
			)
		}) {
			return
		}
		asg.OperatorAccountID = httpx.ClaimsAccountID(c)
		id, err := a.Asset.CreateAssignment(c.Request.Context(), asg)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "asset_assignment", fmt.Sprint(id), map[string]any{
			"op": "assign", "assetId": asg.AssetID, "workerId": asg.WorkerID, "reason": asg.Reason,
		})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "effectiveFrom": asg.EffectiveFrom.Format(time.RFC3339)})
	}
}
