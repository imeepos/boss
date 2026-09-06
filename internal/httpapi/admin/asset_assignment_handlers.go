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

// assignmentReturnHandler POST /asset-assignments/{id}/return:归还(P2-W2-T1 H)。
// 闭合持有段(effective_to=now);重复归还 40900;审计含闭合时间。
func assignmentReturnHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		closedAt, err := a.Asset.ReturnAssignment(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "asset_assignment", c.Param("id"), map[string]any{
			"op": "return", "effectiveTo": closedAt.Format(time.RFC3339),
		})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "effectiveTo": closedAt.Format(time.RFC3339)})
	}
}

// replacementCancelHandler POST /replacements/{id}/cancel:取消换新单(P2-W2-T1 I)。
// 仅 PENDING 可取消 → CANCELLED 终态;非 PENDING 40900;写审计。
func replacementCancelHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		r, err := a.Asset.CancelReplacement(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "replacement", r.ReplacementNo, map[string]any{
			"op": "cancel", "from": "PENDING", "to": "CANCELLED",
		})
		respond(c, apitypes.CodeOK, r)
	}
}
