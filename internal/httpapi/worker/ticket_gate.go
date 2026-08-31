package workerapi

// W 师傅端接单资格闸门:在职 + 负责区域匹配 + 接单设置(在线/接单类型)。

import (
	"errors"
	"math"
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
	if !regionGateOK(c, a, w, tk, "ticket not in worker region") {
		return false
	}
	s, ok := workerGateSettings(c, a, workerID)
	if !ok {
		return false
	}
	if s != nil {
		if !s.Accepting {
			respond(c, apitypes.CodeForbidden, gin.H{"error": "worker not accepting"})
			return false
		}
		if s.AcceptTypes != "" && !strings.Contains(","+s.AcceptTypes+",", ","+portalTicketType+",") {
			respond(c, apitypes.CodeForbidden, gin.H{"error": "ticket type not accepted"})
			return false
		}
	}
	return radiusGateOK(c, a, workerID, tk, s)
}

// regionGateOK 区域子树闸门(祖先或自身):工单区域须落在师傅负责区域集合
// (主区域 ∪ 扩展区域,000175)任一子树内;查询失败按拒绝处理并留错误响应
// (资格判定不允许静默放行)。
func regionGateOK(c *gin.Context, a *app.Application, w *worker.Worker, tk *order.DispatchTicket, rejectMsg string) bool {
	matched, err := a.Worker.MatchedRegionIDs(c.Request.Context(), w, []int64{tk.RegionID})
	if err != nil {
		respondErr(c, err)
		return false
	}
	if !matched[tk.RegionID] {
		respond(c, apitypes.CodeForbidden, gin.H{"error": rejectMsg})
		return false
	}
	return true
}

// workerGateSettings 接单设置读取:未配置(nil/ErrNotFound)按默认在线全类型放行。
func workerGateSettings(c *gin.Context, a *app.Application, workerID int64) (*worker.Settings, bool) {
	if a.WorkerLedger == nil {
		return nil, true
	}
	s, err := a.WorkerLedger.GetSettings(c.Request.Context(), workerID)
	if errors.Is(err, worker.ErrNotFound) {
		return nil, true
	}
	if err != nil {
		respondErr(c, err)
		return nil, false
	}
	return s, true
}

// radiusGateOK 接单半径闸门:接单设置 radiusKm>0 且工单坐标快照(000174)与师傅
// 最新上报位置齐备时,超距即拒;任一前提缺失跳过(与"未设半径=不限距离"同口径)。
func radiusGateOK(c *gin.Context, a *app.Application, workerID int64, tk *order.DispatchTicket, s *worker.Settings) bool {
	if s == nil || s.RadiusKm <= 0 || tk.SiteLat == nil || tk.SiteLng == nil || a.WorkerLocation == nil {
		return true
	}
	loc, err := a.WorkerLocation.LatestLocation(c.Request.Context(), workerID)
	if errors.Is(err, worker.ErrNotFound) {
		return true // 无位置上报不拦截(无法判定≠超距)
	}
	if err != nil {
		respondErr(c, err)
		return false
	}
	if loc == nil {
		return true
	}
	distM := haversineM(loc.Lat, loc.Lng, *tk.SiteLat, *tk.SiteLng)
	if distM <= float64(s.RadiusKm)*1000 {
		return true
	}
	respond(c, apitypes.CodeForbidden, gin.H{
		"error": "ticket out of radius", "distanceM": int(distM), "radiusKm": s.RadiusKm,
	})
	return false
}

// haversineM 两 WGS84 坐标球面距离(米);公里级精度足够接单闸门。
func haversineM(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusM = 6371000.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusM * math.Asin(math.Sqrt(a))
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
	if !regionGateOK(c, a, w, tk, "target worker region mismatch") {
		return nil, false
	}
	return w, true
}
