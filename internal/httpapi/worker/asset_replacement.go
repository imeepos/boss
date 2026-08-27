package workerapi

// W 师傅端换新任务(asset 域 replacements):任务列表 + 现场完成回写。
// adopted note 2026-08-27-replacement-ticket-flow:换新单不经派单工单,本文件
// 是师傅端唯一接入点;完成落既有 worker_replace_logs(ticket_no 落 RPL 号快照)。

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerReplacementCompleteReq 换新完成请求体。
type workerReplacementCompleteReq struct {
	OldEpc string `json:"oldEpc"`
	NewEpc string `json:"newEpc"`
	Result string `json:"result"` // SUCCESS/FAILED
}

// workerReplacementItem 任务列表项(师傅端只读自有字段,资产码展示快照)。
type workerReplacementItem struct {
	ID            int64  `json:"id"`
	ReplacementNo string `json:"replacementNo"`
	AssetID       int64  `json:"assetId"`
	AssetCode     string `json:"assetCode"`
	Reason        string `json:"reason"`
	Priority      string `json:"priority"`
	Status        string `json:"status"`
}

// workerReplacementsHandler GET /replacements:我的换新任务(DOING 执行中)。
func workerReplacementsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		list, err := a.Asset.ListReplacementsByWorker(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]workerReplacementItem, 0, len(list))
		for _, r := range list {
			item := workerReplacementItem{ID: r.ID, ReplacementNo: r.ReplacementNo,
				AssetID: r.AssetID, Reason: r.Reason, Priority: r.Priority, Status: r.Status}
			if ast, err := a.Asset.GetAsset(c.Request.Context(), r.AssetID); err == nil && ast != nil {
				item.AssetCode = ast.AssetCode
			}
			items = append(items, item)
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// workerReplacementCompleteHandler POST /replacements/{id}/complete:现场完成/失败。
// 结果 result=SUCCESS|FAILED;成功落换件流水并回写 DONE,资产状态联动
// (旧件→MAINTENANCE,新件按 EPC 反解→DEPLOYED);失败仅回写 FAILED。
func workerReplacementCompleteHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var req workerReplacementCompleteReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.NewEpc, "newEpc", 64),
				httpx.RequireEnum(req.Result, "result", "SUCCESS", "FAILED"),
			)
		}) {
			return
		}
		workerID, workerName := portalWorker(c)
		ctx := c.Request.Context()

		r, err := a.Asset.GetReplacement(ctx, id)
		if err != nil {
			respondErr(c, err)
			return
		}
		if r.WorkerID != workerID || r.Status != "DOING" {
			respond(c, apitypes.CodeNotFound, nil) // 非本人任务/非执行中,按不存在防探测
			return
		}

		final := "DONE"
		if req.Result == "FAILED" {
			final = "FAILED"
		}

		// 换件流水先行:现场登记是业务事实,后续状态联动失败不回滚登记(留痕可查)。
		if _, err := a.WorkerEvent.AppendReplaceLog(ctx, worker.ReplaceLog{
			WorkerID: workerID, TicketNo: r.ReplacementNo, // 换新单无工单,ticket_no 落 RPL 号展示快照
			OldEpc: req.OldEpc, NewEpc: req.NewEpc, CreatedAt: time.Now(),
		}); err != nil {
			slog.ErrorContext(ctx, "[replacement] REPLACE LOG FAILED",
				"replacementNo", r.ReplacementNo, "workerId", workerID,
				"oldEpc", req.OldEpc, "newEpc", req.NewEpc, "err", err)
			respondErr(c, err)
			return
		}

		if _, err := a.Asset.CompleteReplacement(ctx, id, final); err != nil {
			slog.ErrorContext(ctx, "[replacement] COMPLETE FAILED",
				"replacementNo", r.ReplacementNo, "result", final, "err", err)
			respondErr(c, err)
			return
		}

		applyReplacementAssetTransition(c, a, r, req, final, workerID, workerName)
		httpx.RecordAudit(a, c, "状态变更", "replacement", r.ReplacementNo,
			map[string]any{"result": final, "oldEpc": req.OldEpc, "newEpc": req.NewEpc})
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "status": final})
	}
}

// applyReplacementAssetTransition 完成后资产状态联动;失败仅 WARN 留痕不阻断
// (单已终态、流水已落,联动属补偿动作)。
func applyReplacementAssetTransition(c *gin.Context, a *app.Application, r *asset.Replacement,
	req workerReplacementCompleteReq, final string, workerID int64, workerName string) {
	old := asset.AssetLifecycle{AssetID: r.AssetID, Status: "MAINTENANCE", WorkerID: workerID, WorkerName: workerName, ChangedAt: time.Now()}
	if err := a.Asset.SetAssetStatus(c, r.AssetID, "MAINTENANCE"); err != nil {
		slog.WarnContext(c, "[replacement] OLD ASSET TRANSITION FAILED",
			"assetId", r.AssetID, "replacementNo", r.ReplacementNo, "err", err)
	} else if _, err := a.Asset.AppendLifecycle(c, old); err != nil {
		slog.WarnContext(c, "[replacement] OLD LIFECYCLE FAILED", "assetId", r.AssetID, "err", err)
	}
	if final != "DONE" {
		return
	}
	if newAssetID := assetIDByEpc(c, a, req.NewEpc); newAssetID > 0 {
		if err := a.Asset.SetAssetStatus(c, newAssetID, "DEPLOYED"); err != nil {
			slog.WarnContext(c, "[replacement] NEW ASSET TRANSITION FAILED",
				"assetId", newAssetID, "replacementNo", r.ReplacementNo, "err", err)
		} else if _, err := a.Asset.AppendLifecycle(c, asset.AssetLifecycle{
			AssetID: newAssetID, Status: "DEPLOYED", WorkerID: workerID, WorkerName: workerName, ChangedAt: time.Now()}); err != nil {
			slog.WarnContext(c, "[replacement] NEW LIFECYCLE FAILED", "assetId", newAssetID, "err", err)
		}
	} else {
		slog.WarnContext(c, "[replacement] NEW EPC UNRESOLVED",
			"newEpc", req.NewEpc, "replacementNo", r.ReplacementNo)
	}
}

// assetIDByEpc 按 EPC 唯一键反解资产 id(新件);未命中返回 0。
func assetIDByEpc(c *gin.Context, a *app.Application, epc string) int64 {
	tags, err := a.Asset.ListTags(c.Request.Context())
	if err != nil {
		return 0
	}
	for _, t := range tags {
		if t.EpcCode == epc {
			return t.BoundAssetID
		}
	}
	return 0
}
