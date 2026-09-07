package adminapi

// ODN 质量记录路由(P0-C,迁移 000214):测试证据 append-only + 整改闭环。

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// odnTestReq 测试记录请求体。
type odnTestReq struct {
	ResourceType  string  `json:"resourceType" binding:"required"`
	ResourceRef   string  `json:"resourceRef" binding:"required"`
	TestKind      string  `json:"testKind" binding:"required"`
	Result        string  `json:"result" binding:"required"`
	AttenuationDB float64 `json:"attenuationDb"`
	PowerDBM      float64 `json:"powerDbm"`
	Note          string  `json:"note"`
}

// odnDefectOpenReq 开整改单请求体。
type odnDefectOpenReq struct {
	FacilityCode string `json:"facilityCode" binding:"required"`
	Severity     string `json:"severity" binding:"required"`
	Description  string `json:"description" binding:"required"`
	Note         string `json:"note"`
}

// registerODNQualityRoutes 注册质量路由(menu:odn 门禁;契约 admin/odn.yaml)。
func registerODNQualityRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.POST("/odn/constructions/:id/tests", perm, odnRecordTestHandler(a))
	g.GET("/odn/constructions/:id/tests", perm, odnListTestsHandler(a))
	g.POST("/odn/constructions/:id/defects", perm, odnOpenDefectHandler(a))
	g.GET("/odn/constructions/:id/defects", perm, odnListDefectsHandler(a))
	g.POST("/odn/defects/:defectId/rectify", perm, odnRectifyDefectHandler(a))
	g.POST("/odn/defects/:defectId/verify", perm, odnVerifyDefectHandler(a))
}

// odnRecordTestHandler POST /odn/constructions/{id}/tests:追加测试证据。
func odnRecordTestHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnTestReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		t := odn.QualityTest{ProjectID: id, ResourceType: req.ResourceType, ResourceRef: req.ResourceRef,
			TestKind: req.TestKind, Result: req.Result, AttenuationDB: req.AttenuationDB, PowerDBM: req.PowerDBM,
			Note: req.Note, ReportedBy: httpx.ClaimsAccountID(c)}
		entryID, err := a.ODN.RecordTest(c.Request.Context(), t)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.quality.test", "construction_projects", c.Param("id"),
			map[string]any{"ref": req.ResourceRef, "kind": req.TestKind, "result": req.Result})
		respond(c, apitypes.CodeOK, gin.H{"id": entryID})
	}
}

// odnListTestsHandler GET /odn/constructions/{id}/tests:测试记录列表。
func odnListTestsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		list, err := a.ODN.ListTests(c.Request.Context(), id, limit)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnOpenDefectHandler POST /odn/constructions/{id}/defects:开整改单。
func odnOpenDefectHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnDefectOpenReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		d := odn.QualityDefect{ProjectID: id, FacilityCode: req.FacilityCode, Severity: req.Severity,
			Description: req.Description, Note: req.Note}
		defectID, err := a.ODN.OpenDefect(c.Request.Context(), d, httpx.ClaimsAccountID(c))
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.quality.defect-open", "construction_projects", c.Param("id"),
			map[string]any{"defectId": defectID, "severity": req.Severity, "facility": req.FacilityCode})
		respond(c, apitypes.CodeOK, gin.H{"id": defectID})
	}
}

// odnListDefectsHandler GET /odn/constructions/{id}/defects?status=:整改单列表。
func odnListDefectsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		list, err := a.ODN.ListDefects(c.Request.Context(), id, c.Query("status"), limit)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnRectifyDefectHandler POST /odn/defects/{defectId}/rectify:整改上报。
func odnRectifyDefectHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		defectID, _ := strconv.ParseInt(c.Param("defectId"), 10, 64)
		if err := a.ODN.TransitionDefect(c.Request.Context(), defectID, odn.DefectRectifying, httpx.ClaimsAccountID(c)); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.quality.defect-rectify", "construction_defects", c.Param("defectId"), nil)
		respond(c, apitypes.CodeOK, gin.H{"id": defectID, "status": "RECTIFYING"})
	}
}

// odnVerifyDefectHandler POST /odn/defects/{defectId}/verify:复验关闭。
func odnVerifyDefectHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		defectID, _ := strconv.ParseInt(c.Param("defectId"), 10, 64)
		if err := a.ODN.TransitionDefect(c.Request.Context(), defectID, odn.DefectVerified, httpx.ClaimsAccountID(c)); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.quality.defect-verify", "construction_defects", c.Param("defectId"), nil)
		respond(c, apitypes.CodeOK, gin.H{"id": defectID, "status": "VERIFIED"})
	}
}
