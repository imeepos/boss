package adminapi

// ODN 施工进度上报路由(P0-B,迁移 000213):资源级现场事实,弱网幂等。

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// odnProgressReq 进度上报请求体(clientMsgId 幂等键,坐标可带)。
type odnProgressReq struct {
	FacilityCode string  `json:"facilityCode" binding:"required"`
	DoneQty      float64 `json:"doneQty"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	Note         string  `json:"note"`
	PhotoIDs     []int64 `json:"photoIds"`
	ClientMsgID  string  `json:"clientMsgId" binding:"required"`
}

// registerODNProgressRoutes 注册进度路由(menu:odn 门禁;契约 admin/odn.yaml)。
func registerODNProgressRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.POST("/odn/constructions/:id/progress", perm, odnRecordProgressHandler(a))
	g.GET("/odn/constructions/:id/progress", perm, odnListProgressHandler(a))
}

// odnRecordProgressHandler POST /odn/constructions/{id}/progress:幂等上报。
func odnRecordProgressHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnProgressReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		if req.PhotoIDs == nil {
			req.PhotoIDs = []int64{}
		}
		p := odn.ProgressEntry{ProjectID: id, FacilityCode: req.FacilityCode, DoneQty: req.DoneQty,
			Lat: req.Lat, Lng: req.Lng, Note: req.Note, PhotoIDs: req.PhotoIDs, ClientMsgID: req.ClientMsgID,
			ReportedBy: httpx.ClaimsAccountID(c)}
		entryID, created, err := a.ODN.RecordProgress(c.Request.Context(), p)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.construction.progress", "construction_projects", c.Param("id"),
			map[string]any{"facility": req.FacilityCode, "doneQty": req.DoneQty, "clientMsgId": req.ClientMsgID, "created": created})
		respond(c, apitypes.CodeOK, gin.H{"id": entryID, "created": created})
	}
}

// odnListProgressHandler GET /odn/constructions/{id}/progress:上报列表+清单级进度聚合。
func odnListProgressHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		entries, err := a.ODN.ListProgress(c.Request.Context(), id, limit)
		if err != nil {
			respondErr(c, err)
			return
		}
		summary, err := a.ODN.ItemProgressSummary(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"entries": entries, "items": summary})
	}
}
