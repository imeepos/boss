package adminapi

// 装维队管理 handler(000141):队伍改名/软删/指定队长、成员调队、队伍业绩统计。
// 路由注册见 worker.go registerWorkerRoutes;契约 api/openapi/admin/worker.yaml。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/clock"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerGroupUpdateReq PUT /worker-groups/{groupId}:改名 + 指定队长(leaderId=0 清空)。
type workerGroupUpdateReq struct {
	Name     string `json:"name"`
	LeaderID int64  `json:"leaderId"`
}

// workerUpdateGroupHandler 队伍维护:改名/指定队长(队长须为本队在职成员)。
func workerUpdateGroupHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, ok := httpx.ParsePathParamInt64(c, "groupId")
		if !ok {
			return
		}
		var req workerGroupUpdateReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(httpx.RequireString(req.Name, "name", 64))
		}) {
			return
		}
		g, err := a.WorkerTeam.UpdateGroup(c.Request.Context(), groupID, req.Name, req.LeaderID)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker_group.update", "worker_group", strconv.FormatInt(groupID, 10), gin.H{
			"name": req.Name, "leaderId": req.LeaderID,
		})
		respond(c, apitypes.CodeOK, g)
	}
}

// workerDeleteGroupHandler 队伍软删;仍有在职成员时拒绝。
func workerDeleteGroupHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, ok := httpx.ParsePathParamInt64(c, "groupId")
		if !ok {
			return
		}
		if err := a.WorkerTeam.SoftDeleteGroup(c.Request.Context(), groupID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker_group.delete", "worker_group", strconv.FormatInt(groupID, 10), nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// workerTeamPerformanceHandler GET /worker-groups/{groupId}/performance?period=YYYY-MM:
// 队伍成员业绩统计(完成量/准时率/评分),period 空取当月。
func workerTeamPerformanceHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, ok := httpx.ParsePathParamInt64(c, "groupId")
		if !ok {
			return
		}
		period := c.Query("period")
		if period == "" {
			period = clock.Now().Format("2006-01")
		}
		items, err := a.WorkerTeam.ListTeamPerformances(c.Request.Context(), groupID, period)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"period": period, "items": items})
	}
}

// workerTransferReq POST /workers/{workerId}/transfer:成员调队(落归属台账)。
type workerTransferReq struct {
	GroupID int64  `json:"groupId"`
	Reason  string `json:"reason"`
}

// workerTransferHandler 成员调队:改当前归属 + worker_group_memberships 台账。
func workerTransferHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, ok := httpx.ParsePathParamInt64(c, "workerId")
		if !ok {
			return
		}
		var req workerTransferReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(httpx.RequirePositiveID(req.GroupID, "groupId"))
		}) {
			return
		}
		err := a.WorkerTeam.TransferWorker(c.Request.Context(), worker.Transfer{
			WorkerID: workerID, TargetGroupID: req.GroupID, Reason: req.Reason,
			OperatorAccountID: httpx.ClaimsAccountID(c),
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker.transfer", "worker", strconv.FormatInt(workerID, 10), gin.H{
			"toGroup": req.GroupID, "reason": req.Reason,
		})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
