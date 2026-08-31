package workerapi

// W 师傅端接单资格闸门:在职 + 负责区域匹配 + 接单设置(在线/接单类型)。

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalTicketType 派单工单作业类型(当前仅新装,报障并入派单工单视图)。
const portalTicketType = "INSTALL"

// workerMayAccept 接单/抢单资格闸门:未通过时已写响应并返回 false。
// radiusKm 闸门见 radiusGate(工单坐标快照 000174 起可校验)。
func workerMayAccept(c *gin.Context, a *app.Application, workerID int64, tk *order.DispatchTicket) bool {
	w, err := a.Worker.GetWorker(c.Request.Context(), workerID)
	if err != nil {
		respondErr(c, err)
		return false
	}
	if w.Status != 1 || w.LeftAt != nil {
		respond(c, apitypes.CodeForbidden, gin.H{"error": "worker not active"})
		return false
	}
	if !regionGateOK(c, a, w.RegionID, tk) {
		return false
	}
	return workerSettingsAllow(c, a, workerID)
}

// regionGateOK 区域子树闸门(祖先或自身):工单区域须落在师傅区域子树内;
// 查询失败按拒绝处理并留错误响应(资格判定不允许静默放行)。
func regionGateOK(c *gin.Context, a *app.Application, workerRegionID int64, tk *order.DispatchTicket) bool {
	matched, err := a.Worker.MatchedRegionIDs(c.Request.Context(), workerRegionID, []int64{tk.RegionID})
	if err != nil {
		respondErr(c, err)
		return false
	}
	if !matched[tk.RegionID] {
		respond(c, apitypes.CodeForbidden, gin.H{"error": "ticket not in worker region"})
		return false
	}
	return true
}

// workerSettingsAllow 接单设置闸门:停接单/接单类型不符拒绝;
// 未配置设置按默认(在线、不限类型)放行,与接单设置查询口径一致。
func workerSettingsAllow(c *gin.Context, a *app.Application, workerID int64) bool {
	if a.WorkerLedger == nil {
		return true
	}
	s, err := a.WorkerLedger.GetSettings(c.Request.Context(), workerID)
	if err != nil {
		if errors.Is(err, worker.ErrNotFound) {
			return true
		}
		respondErr(c, err)
		return false
	}
	if s == nil {
		return true
	}
	if !s.Accepting {
		respond(c, apitypes.CodeForbidden, gin.H{"error": "worker not accepting"})
		return false
	}
	if s.AcceptTypes != "" && !strings.Contains(","+s.AcceptTypes+",", ","+portalTicketType+",") {
		respond(c, apitypes.CodeForbidden, gin.H{"error": "ticket type not accepted"})
		return false
	}
	return true
}

// workerTransferTargetOK 师傅端转单目标闸门:目标必须在职且与工单区域匹配
// (师傅自助转单无强制跨区通道,跨区请走后台调度)。
func workerTransferTargetOK(c *gin.Context, a *app.Application, targetID int64, tk *order.DispatchTicket) (*worker.Worker, bool) {
	w, err := a.Worker.GetWorker(c.Request.Context(), targetID)
	if err != nil {
		respondErr(c, err)
		return nil, false
	}
	if w.Status != 1 || w.LeftAt != nil {
		respond(c, apitypes.CodeForbidden, gin.H{"error": "target worker not active"})
		return nil, false
	}
	if !regionGateOK(c, a, w.RegionID, tk) {
		return nil, false
	}
	return w, true
}
