package adminapi

// W 师傅域路由 handler 实现(承接 registerWorkerRoutes 的扁平路由表)。
// 公告 + 站内消息。

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerListNoticesHandler GET /notices:公告列表(含已下架)(worker.yaml /notices)。
func workerListNoticesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkerNotice.ListNotices(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// workerPublishNoticeHandler POST /notices:发布公告。
func workerPublishNoticeHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req publishNoticeReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.Title, "title", 128),
			)
		}) {
			return
		}
		id, err := a.WorkerNotice.CreateNotice(c.Request.Context(), worker.Notice{
			Title: req.Title, Category: req.Category, Active: true,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "notice.publish", "notice", strconv.FormatInt(id, 10), nil)
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// workerToggleNoticeHandler PUT /notices/{noticeId}/toggle:公告上下架切换。
func workerToggleNoticeHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		noticeID, ok := httpx.ParsePathParamInt64(c, "noticeId")
		if !ok {
			return
		}
		if err := a.WorkerNotice.ToggleNotice(c.Request.Context(), noticeID); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// workerListMessagesHandler GET /worker-messages:师傅消息列表。
func workerListMessagesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkerLedger.ListMessages(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// workerSendMessageHandler POST /worker-messages:下发站内消息(即时出现在师傅端消息中心,worker.yaml POST /worker-messages)。
func workerSendMessageHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req sendWorkerMessageReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(req.WorkerID, "workerId"),
				httpx.RequireString(req.Title, "title", 128),
			)
		}) {
			return
		}
		if _, err := a.Worker.GetWorker(c.Request.Context(), req.WorkerID); err != nil {
			respondErr(c, err)
			return
		}
		level := req.Level
		if level == "" {
			level = "INFO"
		}
		id, err := a.WorkerLedger.SendMessage(c.Request.Context(), worker.Message{
			WorkerID: req.WorkerID, Level: level, Title: req.Title, Content: req.Content, SentAt: time.Now(),
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker_message.send", "worker_message", strconv.FormatInt(id, 10), nil)
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}
