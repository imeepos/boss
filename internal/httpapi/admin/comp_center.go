package adminapi

// S2 统一补偿任务/责任队列:领取/转派/重试/回放/关闭/审计。
// 全部操作走 audit_log 留痕;menu:report 门禁。

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/report"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerCompTaskRoutes 注册补偿任务中心路由(menu:report)。
func registerCompTaskRoutes(g *gin.RouterGroup, a *app.Application) {
	ct := g.Group("", requirePerm(a.User, "menu:report"))
	ct.GET("/comp-tasks", compTaskListHandler(a))
	ct.GET("/comp-tasks/:id", compTaskGetHandler(a))
	ct.POST("/comp-tasks/:id/claim", compTaskClaimHandler(a))
	ct.POST("/comp-tasks/:id/transfer", compTaskTransferHandler(a))
	ct.POST("/comp-tasks/:id/retry", compTaskRetryHandler(a))
	ct.POST("/comp-tasks/:id/replay", compTaskReplayHandler(a))
	ct.POST("/comp-tasks/:id/close", compTaskCloseHandler(a))
}

// compTaskService 取服务(空则返回 nil caller 自行降级)。
func compTaskService(a *app.Application) *report.CompTaskService {
	return a.CompTask
}

// compTaskListHandler GET /comp-tasks:列表(过滤+分页)。
func compTaskListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		s := compTaskService(a)
		if s == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		f := report.CompTaskFilter{
			Status:   c.Query("status"),
			Source:   c.Query("source"),
			Priority: c.Query("priority"),
			BizType:  c.Query("bizType"),
		}
		if v := c.Query("assigneeId"); v != "" {
			if id, err := strconv.ParseInt(v, 10, 64); err == nil {
				f.AssigneeID = &id
			}
		}
		f.Offset, _ = strconv.Atoi(c.Query("offset"))
		f.Limit, _ = strconv.Atoi(c.Query("limit"))
		tasks, total, err := s.List(c.Request.Context(), f)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": tasks, "total": total})
	}
}

// compTaskGetHandler GET /comp-tasks/:id:详情(含审计日志)。
func compTaskGetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		s := compTaskService(a)
		if s == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		task, err := s.Get(c.Request.Context(), id)
		if err != nil {
			if errors.Is(err, report.ErrCompTaskNotFound) {
				respond(c, apitypes.CodeNotFound, nil)
				return
			}
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, task)
	}
}

// compTaskClaimHandler POST /comp-tasks/:id/claim:领取任务。
func compTaskClaimHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		s := compTaskService(a)
		if s == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		actorID := httpx.ClaimsAccountID(c)
		actorName := fmt.Sprintf("account_%d", actorID)
		if err := s.Claim(c.Request.Context(), id, actorID, actorName); err != nil {
			if errors.Is(err, report.ErrIllegalStatus) {
				respond(c, apitypes.CodeStateInvalid, nil)
				return
			}
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "comp.claim", "compensation_task", strconv.FormatInt(id, 10), nil)
		respond(c, apitypes.CodeOK, nil)
	}
}

// compTaskTransferHandler POST /comp-tasks/:id/transfer:转派任务。
func compTaskTransferHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		s := compTaskService(a)
		if s == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		var req struct {
			ToID   int64  `json:"toId"`
			ToName string `json:"toName"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.ToID == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		actorID := httpx.ClaimsAccountID(c)
		actorName := fmt.Sprintf("account_%d", actorID)
		if err := s.Transfer(c.Request.Context(), id, actorID, req.ToID, req.ToName, actorName); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "comp.transfer", "compensation_task", strconv.FormatInt(id, 10),
			map[string]any{"toId": req.ToID, "toName": req.ToName})
		respond(c, apitypes.CodeOK, nil)
	}
}

// compTaskRetryHandler POST /comp-tasks/:id/retry:重试(计数+1)。
func compTaskRetryHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		s := compTaskService(a)
		if s == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := s.Retry(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "comp.retry", "compensation_task", strconv.FormatInt(id, 10), nil)
		respond(c, apitypes.CodeOK, nil)
	}
}

// compTaskReplayHandler POST /comp-tasks/:id/replay:回放(关单后重开)。
func compTaskReplayHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		s := compTaskService(a)
		if s == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		task, err := s.Get(c.Request.Context(), id)
		if err != nil {
			if errors.Is(err, report.ErrCompTaskNotFound) {
				respond(c, apitypes.CodeNotFound, nil)
				return
			}
			respondErr(c, err)
			return
		}
		if task.Status != report.TaskStatusClosed {
			respond(c, apitypes.CodeStateInvalid, gin.H{"error": "只能回放已关闭任务"})
			return
		}
		actorID := httpx.ClaimsAccountID(c)
		actorName := fmt.Sprintf("account_%d", actorID)
		task.Status = report.TaskStatusOpen
		task.CloseReason = ""
		task.ClosedAt = nil
		task.ClosedBy = nil
		task.RetryCount = 0
		task.LastRetryAt = nil
		task.AuditLog = report.AppendAudit(task.AuditLog, actorName, "replay", "回放重开")
		if err := s.St.UpdateCompTask(c.Request.Context(), task); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "comp.replay", "compensation_task", strconv.FormatInt(id, 10), nil)
		respond(c, apitypes.CodeOK, nil)
	}
}

// compTaskCloseHandler POST /comp-tasks/:id/close:关闭任务。
func compTaskCloseHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		s := compTaskService(a)
		if s == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		var req struct {
			Reason string `json:"reason"`
		}
		_ = c.ShouldBindJSON(&req)
		actorID := httpx.ClaimsAccountID(c)
		actorName := fmt.Sprintf("account_%d", actorID)
		if err := s.Close(c.Request.Context(), id, actorID, actorName, req.Reason); err != nil {
			if errors.Is(err, report.ErrIllegalStatus) {
				respond(c, apitypes.CodeStateInvalid, nil)
				return
			}
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "comp.close", "compensation_task", strconv.FormatInt(id, 10),
			map[string]any{"reason": req.Reason})
		respond(c, apitypes.CodeOK, nil)
	}
}
