package app

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerWorkerRoutes 注册师傅域路由(承接 api/openapi/admin/worker.yaml)。
func registerWorkerRoutes(g *gin.RouterGroup, a *Application) {
	g.GET("/worker-groups", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.Worker.ListGroups(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/workers", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		list, err := a.Worker.ListWorkers(c.Request.Context(), queryInt64(c, "groupId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/worker-performances", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.WorkerFact.ListPerformances(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/worker-commissions", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.WorkerFact.ListCommissions(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/worker-schedules", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.WorkerFact.ListSchedules(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/worker-materials", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.WorkerEvent.ListMaterials(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/worker-tools", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.WorkerEvent.ListTools(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/worker-feedbacks", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.WorkerEvent.ListFeedbacks(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/asset-returns", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.WorkerEvent.ListAssetReturns(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/worker-messages", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		list, err := a.WorkerLedger.ListMessages(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	// 下发站内消息(即时出现在师傅端消息中心,worker.yaml POST /worker-messages)。
	g.POST("/worker-messages", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		var req sendWorkerMessageReq
		if err := c.ShouldBindJSON(&req); err != nil || req.Title == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		level := req.Level
		if level == "" {
			level = "INFO"
		}
		id, err := a.WorkerLedger.SendMessage(c.Request.Context(), worker.Message{
			WorkerID: req.WorkerID, Level: level, Title: req.Title, Content: req.Content,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "worker_message.send", "worker_message", strconv.FormatInt(id, 10), nil)
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	// 修改接单设置(在线/半径/接单类型),师傅1:1 即时生效(worker.yaml /workers/{id}/settings)。
	g.PUT("/workers/:workerId/settings", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		workerID, _ := strconv.ParseInt(c.Param("workerId"), 10, 64)
		if _, err := a.Worker.GetWorker(c.Request.Context(), workerID); err != nil {
			respondErr(c, err)
			return
		}
		var req workerSettingsReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if _, err := a.WorkerLedger.UpsertSettings(c.Request.Context(), worker.Settings{
			WorkerID: workerID, Accepting: req.Online,
			RadiusKm: req.RadiusKm, AcceptTypes: strings.Join(req.AcceptTypes, ","),
		}); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	// 公告:列表(含已下架)/发布/上下架切换(worker.yaml /notices)。
	g.GET("/notices", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		list, err := a.WorkerNotice.ListNotices(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.POST("/notices", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		var req publishNoticeReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		id, err := a.WorkerNotice.CreateNotice(c.Request.Context(), worker.Notice{
			Title: req.Title, Category: req.Category, Active: true,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "notice.publish", "notice", strconv.FormatInt(id, 10), nil)
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	g.PUT("/notices/:noticeId/toggle", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		noticeID, _ := strconv.ParseInt(c.Param("noticeId"), 10, 64)
		if err := a.WorkerNotice.ToggleNotice(c.Request.Context(), noticeID); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
}

// workerSettingsReq 接单设置请求体(对齐 worker.yaml saveWorkerSettings)。
type workerSettingsReq struct {
	Online      bool     `json:"online"`
	RadiusKm    int16    `json:"radiusKm"`
	AcceptTypes []string `json:"acceptTypes"`
}

// publishNoticeReq 发布公告请求体(title 必填)。
type publishNoticeReq struct {
	Title    string `json:"title" binding:"required"`
	Category string `json:"category"`
}

// sendWorkerMessageReq 下发师傅消息请求体(title 必填;workerId=0 为全员广播语义由端上解释)。
type sendWorkerMessageReq struct {
	WorkerID int64  `json:"workerId"`
	Level    string `json:"level"` // INFO/WARN/URGENT,空取 INFO
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content"`
}
