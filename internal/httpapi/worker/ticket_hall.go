package workerapi

// 任务池(大厅)与抢单(从 ticket.go 拆出,保持单文件 ≤300 行)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/clock"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerHallHandler 任务池:未指派的公开工单,仅含与师傅负责区域匹配的单。
func workerHallHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		w, err := a.Worker.GetWorker(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !hallItemsForWorker(c, a, w) {
			return
		}
	}
}

// hallItemsForWorker 输出任务池:工单须未指派、待派且区域命中师傅任一负责区域(0=不限区域)。
func hallItemsForWorker(c *gin.Context, a *app.Application, w *worker.Worker) bool {
	tickets, err := a.WorkOrder.ListDispatchTickets(c.Request.Context())
	if err != nil {
		respondErr(c, err)
		return false
	}
	inRegion := make(map[string]bool, len(tickets))
	for _, t := range tickets {
		inRegion[t.TicketNo] = w.MatchesRegion(t.RegionID)
	}
	list, err := a.WorkOrder.ListTicketItems(c.Request.Context())
	if err != nil {
		respondErr(c, err)
		return false
	}
	items := make([]gin.H, 0)
	for _, it := range list {
		if it.WorkerID == 0 && it.Status == "PENDING" && inRegion[it.TicketNo] {
			items = append(items, portalTicketOf(it, 0))
		}
	}
	respond(c, apitypes.CodeOK, gin.H{"items": items})
	return true
}

// workerGrabHandler 抢单:先到先得,PENDING 且未指派才可抢。
func workerGrabHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		assignTicketToMe(c, a, c.Param("ticketNo"), true)
	}
}

// nowHM 当前时间 HH:mm(签到/打卡回执)。
func nowHM() string { return clock.Now().Format("15:04") }
