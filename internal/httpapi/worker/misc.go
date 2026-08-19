package workerapi

// W 师傅端门户杂项域(worker/misc.yaml):消息/公告/排障手册/联系调度/安全上报。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerWorkerPortalMiscRoutes 杂项域路由(wauth 组)。
func registerWorkerPortalMiscRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/messages", workerMessagesHandler(a))
	g.POST("/messages/read-all", workerAuditOK(a, "messages-read-all"))
	g.POST("/messages/clear", workerAuditOK(a, "messages-clear"))
	g.GET("/notices", workerNoticesHandler(a))
	g.GET("/help/faq", func(c *gin.Context) {
		respond(c, apitypes.CodeOK, gin.H{"items": []gin.H{}})
	})
	g.GET("/service/messages", func(c *gin.Context) {
		respond(c, apitypes.CodeOK, gin.H{"items": []gin.H{}})
	})
	g.POST("/service/messages", workerAuditOK(a, "service-message-send"))
	g.POST("/safety/checks", workerSafetyCheckHandler(a))
}

// workerMessagesHandler 消息中心(level 枚举 INFO/WARN/URGENT)。
func workerMessagesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		list, err := a.WorkerLedger.ListMessages(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0, len(list))
		for _, m := range list {
			items = append(items, gin.H{
				"level": m.Level, "title": m.Title, "content": m.Content,
				"sentAt": m.SentAt.Format("01-02 15:04"), "read": m.Read,
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// workerNoticesHandler 服务公告:仅 active=true(已上架)对师傅端可见。
func workerNoticesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkerNotice.ListNotices(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0)
		for _, n := range list {
			if n.Active {
				items = append(items, gin.H{
					"noticeId": n.ID, "title": n.Title, "category": n.Category,
					"publishedAt": n.PublishedAt.Format("01-02"),
				})
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// workerSafetyCheckReq 安全作业确认请求体。
type workerSafetyCheckReq struct {
	WorkType  string   `json:"workType" binding:"required"`
	Checklist []string `json:"checklist" binding:"required"`
}

// workerSafetyCheckHandler 安全上报:无 safety_checks 表,审计留痕(缺口见报告)。
func workerSafetyCheckHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req workerSafetyCheckReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "worker_safety", req.WorkType,
			map[string]any{"checklist": req.Checklist})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
