package adminapi

// 勘测任务路由(P-INFRA-1 W7,迁移 000223;menu:odn 门禁;契约 admin/odn.yaml)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// odnSurveyCreateReq 勘测任务创建请求(assignedWorkerId 可带,0=未指派进抢单池)。
type odnSurveyCreateReq struct {
	Title            string `json:"title" binding:"required,max=128"`
	Description      string `json:"description" binding:"max=500"`
	PrvCode          string `json:"prvCode"`
	CityPrefix       string `json:"cityPrefix"`
	GridCode         int64  `json:"gridCode"`
	AssignedWorkerID int64  `json:"assignedWorkerId"`
}

// odnSurveyAssignReq 指派/改派请求。
type odnSurveyAssignReq struct {
	WorkerID int64 `json:"workerId" binding:"required,gt=0"`
}

// registerODNSurveyRoutes 注册勘测任务路由。
func registerODNSurveyRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.GET("/odn/surveys", perm, odnListSurveysHandler(a))
	g.POST("/odn/surveys", perm, odnCreateSurveyHandler(a))
	g.GET("/odn/surveys/:id", perm, odnGetSurveyHandler(a))
	g.POST("/odn/surveys/:id/assign", perm, odnAssignSurveyHandler(a))
	g.POST("/odn/surveys/:id/cancel", perm, odnCancelSurveyHandler(a))
}

func odnListSurveysHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		assignee, _ := strconv.ParseInt(c.Query("workerId"), 10, 64)
		items, err := a.ODN.ListSurveys(c.Request.Context(), c.Query("status"), assignee, limit)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

func odnCreateSurveyHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnSurveyCreateReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		in := odn.SurveyCreateInput{Title: req.Title, Description: req.Description, PrvCode: req.PrvCode,
			CityPrefix: req.CityPrefix, GridCode: req.GridCode, AssignedWorkerID: req.AssignedWorkerID,
			CreatedBy: httpx.ClaimsAccountID(c)}
		t, err := a.ODN.CreateSurvey(c.Request.Context(), in)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.survey.create", "survey_tasks", strconv.FormatInt(t.ID, 10),
			map[string]any{"taskNo": t.TaskNo, "title": t.Title, "assignedWorkerId": req.AssignedWorkerID, "gridCode": req.GridCode})
		respond(c, apitypes.CodeOK, t)
	}
}

func odnGetSurveyHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		t, reports, err := a.ODN.GetSurvey(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"task": t, "reports": reports})
	}
}

func odnAssignSurveyHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnSurveyAssignReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.ODN.AssignSurvey(c.Request.Context(), id, req.WorkerID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.survey.assign", "survey_tasks", c.Param("id"),
			map[string]any{"workerId": req.WorkerID})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func odnCancelSurveyHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.ODN.CancelSurvey(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.survey.cancel", "survey_tasks", c.Param("id"), nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
