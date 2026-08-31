package adminapi

// W 师傅域路由注册(承接 api/openapi/admin/worker.yaml)。
// 全部 handler 实现见 worker_handlers.go;此处只保留扁平路由表。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerWorkerRoutes 注册师傅域路由(承接 api/openapi/admin/worker.yaml)。
func registerWorkerRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/worker-groups", requirePerm(a.User, "menu:order"), workerListGroupsHandler(a))
	g.POST("/worker-groups", requirePerm(a.User, "menu:order"), workerCreateGroupHandler(a))
	// 装维队管理(000141):维护/软删/队长/业绩统计。
	g.PUT("/worker-groups/:groupId", requirePerm(a.User, "menu:order"), workerUpdateGroupHandler(a))
	g.DELETE("/worker-groups/:groupId", requirePerm(a.User, "menu:order"), workerDeleteGroupHandler(a))
	g.GET("/worker-groups/:groupId/performance", requirePerm(a.User, "menu:order"), workerTeamPerformanceHandler(a))
	g.POST("/workers/:workerId/transfer", requirePerm(a.User, "menu:order"), workerTransferHandler(a))

	g.GET("/workers", requirePerm(a.User, "menu:dispatch"), workerListWorkersHandler(a))
	g.POST("/workers", requirePerm(a.User, "menu:dispatch"), workerCreateHandler(a))
	g.GET("/workers/:workerId", requirePerm(a.User, "menu:dispatch"), workerGetWorkerHandler(a))
	g.GET("/workers/:workerId/location", requirePerm(a.User, "menu:dispatch"), workerLatestLocationHandler(a))
	g.PUT("/workers/:workerId/settings", requirePerm(a.User, "menu:dispatch"), workerUpdateSettingsHandler(a))
	g.PUT("/workers/:workerId/password", requirePerm(a.User, "menu:dispatch"), workerSetPasswordHandler(a))

	g.GET("/worker-performances", requirePerm(a.User, "menu:order"), workerListPerformancesHandler(a))
	g.GET("/worker-commissions", requirePerm(a.User, "menu:order"), workerListCommissionsHandler(a))
	g.GET("/worker-schedules", requirePerm(a.User, "menu:order"), workerListSchedulesHandler(a))
	g.GET("/worker-materials", requirePerm(a.User, "menu:order"), workerListMaterialsHandler(a))
	g.GET("/worker-tools", requirePerm(a.User, "menu:order"), workerListToolsHandler(a))
	g.GET("/worker-feedbacks", requirePerm(a.User, "menu:order"), workerListFeedbacksHandler(a))
	g.POST("/worker-feedbacks/:feedbackId/review", requirePerm(a.User, "menu:order"), workerReviewFeedbackHandler(a))

	g.GET("/asset-returns", requirePerm(a.User, "menu:order"), workerListAssetReturnsHandler(a))
	g.POST("/asset-returns/:returnId/confirm", requirePerm(a.User, "menu:order"), workerConfirmAssetReturnHandler(a))

	g.GET("/worker-messages", requirePerm(a.User, "menu:dispatch"), workerListMessagesHandler(a))
	g.POST("/worker-messages", requirePerm(a.User, "menu:dispatch"), workerSendMessageHandler(a))

	g.GET("/notices", requirePerm(a.User, "menu:dispatch"), workerListNoticesHandler(a))
	g.POST("/notices", requirePerm(a.User, "menu:dispatch"), workerPublishNoticeHandler(a))
	g.PUT("/notices/:noticeId/toggle", requirePerm(a.User, "menu:dispatch"), workerToggleNoticeHandler(a))
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
