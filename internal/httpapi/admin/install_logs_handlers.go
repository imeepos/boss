package adminapi

// 施工回单 handler(仅 admin 端 GET 列出;POST/Arrive 由 worker 端注册,见 install_logs_worker.go)。

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// installBoardTicketsHandler 施工看板·全量派单工单(含到场打卡事实 arrivedAt/arriveLat/arriveLng)。
func installBoardTicketsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkOrder.ListDispatchTickets(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func installLogListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticketID := queryInt64(c, "ticketId")
		if ticketID <= 0 {
			respondErr(c, fmt.Errorf("ticketId required"))
			return
		}
		list, err := a.WorkOrder.ListInstallLogs(c.Request.Context(), ticketID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}
