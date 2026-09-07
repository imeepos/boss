package adminapi

// ROW/PECE 许可单路由(P-INFRA-1 W4,迁移 000211;审查 F3;契约 admin/odn.yaml)。
// 状态机/开工门控规则见 internal/domain/odn/permit.go+pg_permit*.go;写操作全部入审计。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerODNPermitRoutes 注册许可单路由(门禁 menu:permits,与 menu.def key 同源,先例 menu:grid-investment;
// 不复用组级 menu:odn:页面与接口权限一一对应,授权对齐三方对账门禁)。
func registerODNPermitRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:permits")
	g.GET("/odn/permits", perm, odnListPermitsHandler(a))
	g.POST("/odn/permits", perm, odnCreatePermitHandler(a))
	g.GET("/odn/permits/:id", perm, odnGetPermitHandler(a))
	g.PUT("/odn/permits/:id", perm, odnUpdatePermitHandler(a))
	g.POST("/odn/permits/:id/transition", perm, odnPermitTransitionHandler(a))
	g.POST("/odn/permits/:id/link", perm, odnPermitLinkHandler(a))
	g.POST("/odn/permits/:id/unlink", perm, odnPermitUnlinkHandler(a))
	g.GET("/odn/constructions/:id/permits", perm, odnProjectPermitsHandler(a))
	g.POST("/odn/constructions/:id/permits-check", perm, odnProjectPermitCheckHandler(a))
}

// odnPermitCreateReq 许可单新建请求体(关联项目/设施/资源链可选)。
type odnPermitCreateReq struct {
	Kind          string  `json:"kind" binding:"required,oneof=ROW PECE"`
	Title         string  `json:"title"`
	ApprovalNo    string  `json:"approvalNo"`
	Authority     string  `json:"authority"`
	ValidFrom     string  `json:"validFrom"`
	ValidUntil    string  `json:"validUntil"`
	ProjectID     int64   `json:"projectId"`
	FacilityCode  string  `json:"facilityCode"`
	ChainID       int64   `json:"chainId"`
	AttachmentIDs []int64 `json:"attachmentIds"`
	Note          string  `json:"note"`
}

// odnPermitTransitionReq 状态流转请求体(批准须 approvalNo+validUntil;驳回/退回须 reason)。
type odnPermitTransitionReq struct {
	To         string `json:"to" binding:"required"`
	Reason     string `json:"reason"`
	ApprovalNo string `json:"approvalNo"`
	ValidFrom  string `json:"validFrom"`
	ValidUntil string `json:"validUntil"`
}

// odnPermitLinkReq 关联项目请求体。
type odnPermitLinkReq struct {
	ProjectID int64 `json:"projectId" binding:"required,gt=0"`
}

// odnCreatePermitHandler POST /odn/permits:新建许可单(初始状态随 kind)。
func odnCreatePermitHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnPermitCreateReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		p := odn.Permit{Kind: req.Kind, Title: req.Title, ApprovalNo: req.ApprovalNo,
			Authority: req.Authority, ValidFrom: req.ValidFrom, ValidUntil: req.ValidUntil,
			ProjectID: req.ProjectID, FacilityCode: req.FacilityCode, ChainID: req.ChainID,
			AttachmentIDs: req.AttachmentIDs, Note: req.Note}
		st, err := a.ODN.CreatePermit(c.Request.Context(), p, httpx.ClaimsAccountID(c))
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.permit.create", "odn_permits", strconv.FormatInt(st.ID, 10),
			map[string]any{"permitNo": st.PermitNo, "kind": st.Kind, "projectId": req.ProjectID})
		respond(c, apitypes.CodeOK, st)
	}
}

// odnListPermitsHandler GET /odn/permits?kind=&status=&projectId=&unlinked=&limit=。
func odnListPermitsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		projectID, _ := strconv.ParseInt(c.Query("projectId"), 10, 64)
		list, err := a.ODN.ListPermits(c.Request.Context(), c.Query("kind"), c.Query("status"),
			projectID, c.Query("unlinked") == "1", limit)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnGetPermitHandler GET /odn/permits/{id}:详情(证照档案要素)。
func odnGetPermitHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		p, err := a.ODN.GetPermit(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		if p == nil {
			respond(c, apitypes.CodeNotFound, gin.H{"reason": "permit not found"})
			return
		}
		respond(c, apitypes.CodeOK, p)
	}
}

// odnUpdatePermitHandler PUT /odn/permits/{id}:证照档案要素补录(补件语义,状态不变)。
func odnUpdatePermitHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		var req odnPermitCreateReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		p := odn.Permit{Title: req.Title, ApprovalNo: req.ApprovalNo, Authority: req.Authority,
			ValidFrom: req.ValidFrom, ValidUntil: req.ValidUntil, FacilityCode: req.FacilityCode,
			ChainID: req.ChainID, AttachmentIDs: req.AttachmentIDs, Note: req.Note}
		if err := a.ODN.UpdatePermitArchive(c.Request.Context(), id, p); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.permit.update-archive", "odn_permits", c.Param("id"),
			map[string]any{"approvalNo": req.ApprovalNo, "validUntil": req.ValidUntil, "attachments": len(req.AttachmentIDs)})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// odnPermitTransitionHandler POST /odn/permits/{id}/transition:状态流转(转移表域内校验)。
func odnPermitTransitionHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		var req odnPermitTransitionReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		operator := httpx.ClaimsAccountID(c)
		st, err := a.ODN.TransitionPermit(c.Request.Context(), id, operator, req.To,
			req.Reason, req.ApprovalNo, req.ValidFrom, req.ValidUntil)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.permit.transition", "odn_permits", c.Param("id"),
			map[string]any{"to": req.To, "from": st.Status, "reason": req.Reason, "approvalNo": req.ApprovalNo})
		respond(c, apitypes.CodeOK, st)
	}
}

// odnPermitLinkHandler POST /odn/permits/{id}/link:关联施工项目(单号快照)。
func odnPermitLinkHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		var req odnPermitLinkReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if err := a.ODN.LinkPermitProject(c.Request.Context(), id, req.ProjectID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.permit.link", "odn_permits", c.Param("id"),
			map[string]any{"projectId": req.ProjectID})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// odnPermitUnlinkHandler POST /odn/permits/{id}/unlink:解除项目关联。
func odnPermitUnlinkHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		if err := a.ODN.UnlinkPermitProject(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.permit.unlink", "odn_permits", c.Param("id"), nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// odnProjectPermitsHandler GET /odn/constructions/{id}/permits:项目关联许可清单。
func odnProjectPermitsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		list, err := a.ODN.ListProjectPermits(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnProjectPermitCheckHandler POST /odn/constructions/{id}/permits-check:开工门控预检
// (干跑,不落状态;供前端与验收断言缺失明细)。
func odnProjectPermitCheckHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		rep, err := a.ODN.CheckProjectPermits(c.Request.Context(), id)
		if err != nil && rep == nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, rep)
	}
}
