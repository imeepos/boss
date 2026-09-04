package workerapi

// W 师傅端到点签到(环节9,SLA 计时起点):GPS ≤100m 闸门读 dispatch_tickets
// 站点坐标快照(000174,派单时刻自 addresses.geom 物化)。
// 无坐标工单不再静默放行(2026-09-04 任务A-f):显式降级口径写入响应+审计,
// 有坐标而请求缺定位 → 42200 拒绝,超距 → 40900 拒绝。

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// checkinRadiusM 签到 GPS 闸门半径(到点签到口径,routes_gen「≤100m」)。
const checkinRadiusM = 100.0

// workerCheckinReq 签到定位入参(WGS84;可选体,老版本端不带体)。
type workerCheckinReq struct {
	Lat *float64 `json:"lat"`
	Lng *float64 `json:"lng"`
}

// workerCheckinHandler 到点签到:GPS 闸门 + 显式降级口径。
func workerCheckinHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, _, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !workerOwnedTicket(c, tk) {
			return
		}
		var req workerCheckinReq
		_ = c.ShouldBindJSON(&req) // 缺体视为未带定位
		gps, ok := checkinGPSGate(c, tk, req)
		if !ok {
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "worker_ticket", tk.TicketNo,
			map[string]any{"action": "checkin", "gpsCheck": gps})
		respond(c, apitypes.CodeOK, gin.H{"checkedInAt": nowHM(), "gpsCheck": gps})
	}
}

// checkinGPSGate 签到定位闸门:返回 (口径, 是否放行);拒绝路径已回写响应。
func checkinGPSGate(c *gin.Context, tk *order.DispatchTicket, req workerCheckinReq) (string, bool) {
	if tk.SiteLat == nil || tk.SiteLng == nil {
		// 无坐标工单:显式降级(不静默),口径随响应与审计留痕。
		log.Printf("[worker-checkin] DEGRADED ticket=%s site coords missing, gps gate skipped", tk.TicketNo)
		return "SKIPPED_NO_SITE_COORDS", true
	}
	if req.Lat == nil || req.Lng == nil {
		respond(c, apitypes.CodeInvalidParam, gin.H{"reason": "该工单已配置站点坐标,签到必须携带定位(lat/lng)"})
		return "", false
	}
	distM := haversineM(*req.Lat, *req.Lng, *tk.SiteLat, *tk.SiteLng)
	if distM > checkinRadiusM {
		respond(c, apitypes.CodeConflict, gin.H{
			"reason":    fmt.Sprintf("签到点距站点 %.0fm,超出 %.0fm 闸门", distM, checkinRadiusM),
			"distanceM": int(distM),
		})
		return "", false
	}
	return "OK", true
}
