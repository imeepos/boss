package adminapi

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerWorkerRoutes 注册师傅域路由(承接 api/openapi/admin/worker.yaml)。
func registerWorkerRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/worker-groups", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.Worker.ListGroups(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	// 新建班组(承接 admin 后台基础数据维护;code 公司内唯一)。
	g.POST("/worker-groups", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		var req workerGroupCreateReq
		if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" || req.Name == "" || req.LegalEntityID <= 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		id, err := a.Worker.CreateGroup(c.Request.Context(), worker.Group{
			LegalEntityID: req.LegalEntityID,
			Code:          req.Code,
			Name:          req.Name,
			LeaderID:      req.LeaderID,
			LeaderName:    req.LeaderName,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker_group.create", "worker_group", strconv.FormatInt(id, 10), nil)
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	g.GET("/workers", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		list, err := a.Worker.ListWorkers(c.Request.Context(), queryInt64(c, "groupId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	// 师傅详情(worker.yaml GET /workers/{workerId})。
	g.GET("/workers/:workerId", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		workerID, _ := strconv.ParseInt(c.Param("workerId"), 10, 64)
		w, err := a.Worker.GetWorker(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, w)
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

	// 差评复核(worker.yaml POST /worker-feedbacks/{feedbackId}/review)。
	g.POST("/worker-feedbacks/:feedbackId/review", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		feedbackID, _ := strconv.ParseInt(c.Param("feedbackId"), 10, 64)
		if err := a.WorkerEvent.ReviewFeedback(c.Request.Context(), feedbackID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker_feedback.review", "worker_feedback", c.Param("feedbackId"), nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	g.GET("/asset-returns", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.WorkerEvent.ListAssetReturns(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	// 确认返库(worker.yaml POST /asset-returns/{returnId}/confirm)。
	g.POST("/asset-returns/:returnId/confirm", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		returnID, _ := strconv.ParseInt(c.Param("returnId"), 10, 64)
		if err := a.WorkerEvent.ConfirmAssetReturn(c.Request.Context(), returnID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "asset_return.confirm", "asset_return", c.Param("returnId"), nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
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
		if err := c.ShouldBindJSON(&req); err != nil || req.Title == "" || req.WorkerID <= 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
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
		httpx.RecordAudit(a, c, "notice.publish", "notice", strconv.FormatInt(id, 10), nil)
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

// workerSettingsReq 修改接单设置请求体(与师傅端 settings 端点同形,契约独立演进)。
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
