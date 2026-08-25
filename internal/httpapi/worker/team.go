package workerapi

// 装维队队长业绩视图(000141):GET /team/performance。
// 仅队长(leader_id 指向本人)可见;非队长返回 40300,端上据此隐藏入口。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/clock"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerTeamPerformanceHandler 队长查看本队成员月度业绩统计。
func workerTeamPerformanceHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		g, err := a.WorkerTeam.GroupOfLeader(c.Request.Context(), workerID)
		if err != nil {
			respond(c, apitypes.CodeForbidden, nil)
			return
		}
		period := c.Query("period")
		if period == "" {
			period = clock.Now().Format("2006-01")
		}
		items, err := a.WorkerTeam.ListTeamPerformances(c.Request.Context(), g.ID, period)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"period": period,
			"group":  gin.H{"id": g.ID, "name": g.Name, "memberCount": g.MemberCount},
			"items":  items,
		})
	}
}
