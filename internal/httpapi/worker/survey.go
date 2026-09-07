package workerapi

// 勘测任务师傅端(P-INFRA-1 W7,迁移 000223;契约 worker/survey.yaml):
// 可见集=指派给自己的+未指派抢单池;接单/回填须本人身份,回填 append-only 幂等。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

type workerSurveyReportReq struct {
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	FacilityNote string  `json:"facilityNote"`
	Suggestion   string  `json:"suggestion" binding:"required"`
	PhotoIDs     []int64 `json:"photoIds"`
	ClientMsgID  string  `json:"clientMsgId" binding:"required"`
}

func registerWorkerSurveyRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/surveys", workerSurveyListHandler(a))
	g.GET("/surveys/:id", workerSurveyDetailHandler(a))
	g.POST("/surveys/:id/accept", workerSurveyAcceptHandler(a))
	g.POST("/surveys/:id/reports", workerSurveyReportHandler(a))
}

// workerSurveyListHandler GET /surveys?status=:可见集(取消单不可见)。
func workerSurveyListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		items, err := a.ODN.ListSurveysForWorker(c.Request.Context(), workerID, c.Query("status"), limit)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

func workerSurveyDetailHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		t, reports, err := a.ODN.GetSurvey(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		if t.AssignedWorkerID > 0 && t.AssignedWorkerID != workerID {
			respondErr(c, odn.ErrSurveyNotAssignee)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"task": t, "reports": reports})
	}
}

func workerSurveyAcceptHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, name := portalWorker(c)
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.ODN.AcceptSurvey(c.Request.Context(), id, workerID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker.survey.accept", "survey_tasks", c.Param("id"),
			map[string]any{"workerId": workerID, "workerName": name})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func workerSurveyReportHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, name := portalWorker(c)
		var req workerSurveyReportReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if req.PhotoIDs == nil {
			req.PhotoIDs = []int64{}
		}
		r := odn.SurveyReport{TaskID: id, WorkerID: workerID, Lat: req.Lat, Lng: req.Lng,
			FacilityNote: req.FacilityNote, Suggestion: req.Suggestion, PhotoIDs: req.PhotoIDs,
			ClientMsgID: req.ClientMsgID}
		reportID, created, err := a.ODN.AddSurveyReport(c.Request.Context(), r)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker.survey.report", "survey_tasks", c.Param("id"),
			map[string]any{"workerId": workerID, "workerName": name, "reportId": reportID,
				"suggestion": req.Suggestion, "clientMsgId": req.ClientMsgID, "created": created})
		respond(c, apitypes.CodeOK, gin.H{"id": reportID, "created": created})
	}
}
