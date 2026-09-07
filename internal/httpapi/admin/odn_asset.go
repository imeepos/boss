package adminapi

// ODN 资产化转固路由(P-INFRA-1 W8,迁移 000215;契约 api/openapi/admin/odn.yaml)。
// 桥表软引用裁定 adopted 2026-09-07-odn-asset-capitalization;写操作全部入审计。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerODNAssetRoutes 注册资产化与材料出库路由(menu:odn 门禁)。
func registerODNAssetRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.POST("/odn/assets/registrations", perm, odnCreateRegistrationHandler(a))
	g.GET("/odn/assets/registrations", perm, odnListRegistrationsHandler(a))
	g.POST("/odn/assets/registrations/:id/reverse", perm, odnReverseRegistrationHandler(a))
	g.POST("/odn/material-issues", perm, odnCreateIssueHandler(a))
	g.GET("/odn/material-issues", perm, odnListIssuesHandler(a))
	g.POST("/odn/material-issues/:id/confirm", perm, odnConfirmIssueHandler(a))
	g.POST("/odn/material-issues/:id/cancel", perm, odnCancelIssueHandler(a))
}

// odnRegistrationReq 资产化登记请求体(实体二选一;CONSTRUCTION 来源 projectId 必填)。
type odnRegistrationReq struct {
	EntityKind   string  `json:"entityKind" binding:"required"`
	FacilityCode string  `json:"facilityCode"`
	DeviceID     int64   `json:"deviceId"`
	AssetID      int64   `json:"assetId" binding:"required,min=1"`
	SourceKind   string  `json:"sourceKind" binding:"required"`
	ProjectID    int64   `json:"projectId"`
	ValueAmount  float64 `json:"valueAmount"`
	Remark       string  `json:"remark" binding:"max=255"`
}

// odnCreateRegistrationHandler POST /odn/assets/registrations:建凭证+资产置 DEPLOYED。
func odnCreateRegistrationHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnRegistrationReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		operator := httpx.ClaimsAccountID(c)
		reg, err := a.ODN.CreateRegistration(c.Request.Context(), req.EntityKind, req.FacilityCode,
			req.DeviceID, req.AssetID, req.SourceKind, req.ProjectID, req.ValueAmount, req.Remark, operator)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.assetreg.create", "odn_asset_registrations", strconv.FormatInt(reg.ID, 10),
			map[string]any{"regNo": reg.RegistrationNo, "entityKind": req.EntityKind,
				"facilityCode": req.FacilityCode, "deviceId": req.DeviceID,
				"assetId": req.AssetID, "sourceKind": req.SourceKind, "valueAmount": req.ValueAmount})
		respond(c, apitypes.CodeOK, reg)
	}
}

// odnListRegistrationsHandler GET /odn/assets/registrations:凭证列表(entityRef=设施码或设备id)。
func odnListRegistrationsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
		list, err := a.ODN.ListRegistrations(c.Request.Context(), c.Query("entityKind"),
			c.Query("entityRef"), c.Query("status"), limit)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnReverseRegistrationHandler POST /odn/assets/registrations/{id}/reverse:冲销(原因必填)。
func odnReverseRegistrationHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		operator := httpx.ClaimsAccountID(c)
		var req odnReasonReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if err := a.ODN.ReverseRegistration(c.Request.Context(), id, operator, req.Reason); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.assetreg.reverse", "odn_asset_registrations", c.Param("id"),
			map[string]any{"to": "REVERSED", "reason": req.Reason})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": "REVERSED"})
	}
}

// odnReasonReq 冲销/取消类原因请求体。
type odnReasonReq struct {
	Reason string `json:"reason" binding:"required,max=255"`
}

// odnIssueReq 材料出库请求体(逐台资产;成本归集归 W9)。
type odnIssueReq struct {
	ProjectID int64   `json:"projectId" binding:"required,min=1"`
	AssetIDs  []int64 `json:"assetIds" binding:"required,min=1,max=200"`
	Remark    string  `json:"remark" binding:"max=255"`
}

// odnCreateIssueHandler POST /odn/material-issues:建出库单(OPEN;不动资产状态)。
func odnCreateIssueHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnIssueReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		operator := httpx.ClaimsAccountID(c)
		issue, err := a.ODN.CreateIssue(c.Request.Context(), req.ProjectID, req.AssetIDs, req.Remark, operator)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.issue.create", "odn_material_issues", strconv.FormatInt(issue.ID, 10),
			map[string]any{"issueNo": issue.IssueNo, "projectId": req.ProjectID, "assets": len(req.AssetIDs)})
		respond(c, apitypes.CodeOK, issue)
	}
}

// odnListIssuesHandler GET /odn/material-issues:出库单列表(projectId/status 过滤)。
func odnListIssuesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID, _ := strconv.ParseInt(c.Query("projectId"), 10, 64)
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		list, err := a.ODN.ListIssues(c.Request.Context(), projectID, c.Query("status"), limit)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnConfirmIssueHandler POST /odn/material-issues/{id}/confirm:出库确认(资产→IN_TRANSIT)。
func odnConfirmIssueHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		operator := httpx.ClaimsAccountID(c)
		if err := a.ODN.ConfirmIssue(c.Request.Context(), id, operator); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.issue.confirm", "odn_material_issues", c.Param("id"),
			map[string]any{"to": "CONFIRMED", "assetStatus": "IN_TRANSIT"})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": "CONFIRMED"})
	}
}

// odnCancelIssueHandler POST /odn/material-issues/{id}/cancel:取消(已出库则退库回 IN_STOCK)。
func odnCancelIssueHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		operator := httpx.ClaimsAccountID(c)
		if err := a.ODN.CancelIssue(c.Request.Context(), id, operator); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.issue.cancel", "odn_material_issues", c.Param("id"),
			map[string]any{"to": "CANCELLED"})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": "CANCELLED"})
	}
}
