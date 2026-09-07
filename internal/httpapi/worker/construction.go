package workerapi

// 施工进度师傅端上报(P-INFRA-1 W7,F5a;契约 worker/construction.yaml):
// BUILDING 项目列表/上报上下文;进度上报复用 admin 同表(000213),上报人代次 WORKER(000224),
// 记录只增不改,clientMsgId 幂等不重复计量。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

type workerProgressReq struct {
	FacilityCode string  `json:"facilityCode" binding:"required"`
	DoneQty      float64 `json:"doneQty"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	Note         string  `json:"note"`
	PhotoIDs     []int64 `json:"photoIds"`
	ClientMsgID  string  `json:"clientMsgId" binding:"required"`
}

func registerWorkerConstructionRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/constructions", workerConstructionListHandler(a))
	g.GET("/constructions/:id", workerConstructionDetailHandler(a))
	g.POST("/constructions/:id/progress", workerConstructionProgressHandler(a))
}

func workerConstructionListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		items, err := a.ODN.ListBuildingProjects(c.Request.Context(), limit)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

func workerConstructionDetailHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		project, items, err := a.ODN.GetProjectWorkerView(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"project": project, "items": items})
	}
}

func workerConstructionProgressHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, name := portalWorker(c)
		var req workerProgressReq
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
		p := odn.ProgressEntry{ProjectID: id, FacilityCode: req.FacilityCode, DoneQty: req.DoneQty,
			Lat: req.Lat, Lng: req.Lng, Note: req.Note, PhotoIDs: req.PhotoIDs, ClientMsgID: req.ClientMsgID,
			ReporterType: odn.ProgressReporterWorker, ReportedBy: workerID}
		entryID, created, err := a.ODN.RecordProgress(c.Request.Context(), p)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker.construction.progress", "construction_projects", c.Param("id"),
			map[string]any{"workerId": workerID, "workerName": name, "facility": req.FacilityCode,
				"doneQty": req.DoneQty, "clientMsgId": req.ClientMsgID, "created": created})
		respond(c, apitypes.CodeOK, gin.H{"id": entryID, "created": created})
	}
}
