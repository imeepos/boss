package adminapi

// ODN 施工项目路由(P6,T9,迁移 000199;docs/plan/odn-lifecycle-roadmap.md)。
// 状态机/竣工回填规则见 internal/domain/odn/construction.go+pg_construction.go。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// odnProjectReq 施工单新建请求体。
type odnProjectReq struct {
	ProjNo     string `json:"projNo" binding:"required,max=32"`
	Name       string `json:"name"`
	PrvCode    string `json:"prvCode"`
	CityPrefix string `json:"cityPrefix"`
}

// odnProjectItemReq 明细追加请求体。
type odnProjectItemReq struct {
	FacilityCode string `json:"facilityCode" binding:"required"`
}

// odnProjectAcceptReq 竣工验收请求体。
type odnProjectAcceptReq struct {
	Note string `json:"note"`
}

// registerODNConstructionRoutes 注册施工单路由(menu:odn 门禁;契约 admin/odn.yaml)。
func registerODNConstructionRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.POST("/odn/constructions", perm, odnCreateProjectHandler(a))
	g.GET("/odn/constructions", perm, odnListProjectsHandler(a))
	g.GET("/odn/constructions/:id", perm, odnGetProjectHandler(a))
	g.POST("/odn/constructions/:id/items", perm, odnAddProjectItemHandler(a))
	g.POST("/odn/constructions/:id/start", perm, odnStartProjectHandler(a))
	g.POST("/odn/constructions/:id/accept", perm, odnAcceptProjectHandler(a))
}

// odnCreateProjectHandler POST /odn/constructions:新建施工单。
func odnCreateProjectHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnProjectReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		p := odn.Construction{ProjNo: req.ProjNo, Name: req.Name, PrvCode: req.PrvCode, CityPrefix: req.CityPrefix}
		if err := a.ODN.CreateProject(c.Request.Context(), p); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.construction.create", "construction_projects", req.ProjNo, nil)
		respond(c, apitypes.CodeOK, gin.H{"projNo": req.ProjNo})
	}
}

// odnListProjectsHandler GET /odn/constructions?limit=:施工单列表。
func odnListProjectsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		list, err := a.ODN.ListProjects(c.Request.Context(), limit)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnGetProjectHandler GET /odn/constructions/{id}:详情+明细。
func odnGetProjectHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		p, err := a.ODN.GetProject(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		if p == nil {
			respond(c, apitypes.CodeNotFound, gin.H{"reason": "project not found"})
			return
		}
		items, err := a.ODN.ListProjectItems(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"project": p, "items": items})
	}
}

// odnAddProjectItemHandler POST /odn/constructions/{id}/items:追加明细。
func odnAddProjectItemHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnProjectItemReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		if err := a.ODN.AddProjectItem(c.Request.Context(), id, req.FacilityCode); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.construction.add-item", "construction_projects", c.Param("id"), map[string]any{"facility": req.FacilityCode})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// odnStartProjectHandler POST /odn/constructions/{id}/start:开工,PLANNED 设施批量转 IN_BUILD。
func odnStartProjectHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		flipped, err := a.ODN.StartProject(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.construction.start", "construction_projects", c.Param("id"), map[string]any{"flipped": flipped})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": "BUILDING", "flipped": flipped})
	}
}

// odnAcceptProjectHandler POST /odn/constructions/{id}/accept:竣工验收,设施批量回填 IN_SERVICE。
func odnAcceptProjectHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnProjectAcceptReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		operator := httpx.ClaimsAccountID(c)
		flipped, err := a.ODN.AcceptProject(c.Request.Context(), id, operator, req.Note)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.construction.accept", "construction_projects", c.Param("id"), map[string]any{"flipped": flipped, "note": req.Note})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": "ACCEPTED", "flipped": flipped})
	}
}
