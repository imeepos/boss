package adminapi

// W 师傅域路由 handler 实现(承接 registerWorkerRoutes 的扁平路由表)。
// 师傅事实 / 事件台账:绩效、提成、排班、物料、工具、评价、资产返库。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerListPerformancesHandler GET /worker-performances:师傅绩效列表。
func workerListPerformancesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkerFact.ListPerformances(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// workerListCommissionsHandler GET /worker-commissions:师傅提成列表。
func workerListCommissionsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkerFact.ListCommissions(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// workerListSchedulesHandler GET /worker-schedules:师傅排班列表。
func workerListSchedulesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkerFact.ListSchedules(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// workerListMaterialsHandler GET /worker-materials:师傅物料使用列表。
func workerListMaterialsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkerEvent.ListMaterials(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// workerListToolsHandler GET /worker-tools:师傅工具使用列表。
func workerListToolsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkerEvent.ListTools(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// workerListFeedbacksHandler GET /worker-feedbacks:师傅评价列表。
func workerListFeedbacksHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkerEvent.ListFeedbacks(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// workerReviewFeedbackHandler POST /worker-feedbacks/{feedbackId}/review:差评复核(worker.yaml POST /worker-feedbacks/{feedbackId}/review)。
func workerReviewFeedbackHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		feedbackID, ok := httpx.ParsePathParamInt64(c, "feedbackId")
		if !ok {
			return
		}
		if err := a.WorkerEvent.ReviewFeedback(c.Request.Context(), feedbackID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker_feedback.review", "worker_feedback", c.Param("feedbackId"), nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// workerListAssetReturnsHandler GET /asset-returns:资产返库列表。
func workerListAssetReturnsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkerEvent.ListAssetReturns(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// workerConfirmAssetReturnHandler POST /asset-returns/{returnId}/confirm:确认返库(worker.yaml POST /asset-returns/{returnId}/confirm)。
func workerConfirmAssetReturnHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		returnID, ok := httpx.ParsePathParamInt64(c, "returnId")
		if !ok {
			return
		}
		if err := a.WorkerEvent.ConfirmAssetReturn(c.Request.Context(), returnID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "asset_return.confirm", "asset_return", c.Param("returnId"), nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
