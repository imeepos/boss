package adminapi

// 企业员工后台录入 handler(000173,路由注册见 org.go):
// 后台按企业维度录入员工登录信息(工号/登录名/密码/角色),menu:company 保护。
// 与企业工作台自助建号(partner_staff handler)平行;密码只在录入/重置请求出现,审计不落明文。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// entityStaffCreateReq 后台录入企业员工请求体(工号可空,角色限 partner_admin/partner_staff)。
type entityStaffCreateReq struct {
	StaffNo  string `json:"staffNo"`
	Username string `json:"username"`
	Password string `json:"password"`
	RealName string `json:"realName"`
	Phone    string `json:"phone"`
	RoleCode string `json:"roleCode"`
}

// orgListEntityStaffHandler GET /legal-entities/{id}/staff:企业员工列表(含工号)。
func orgListEntityStaffHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		entityID, ok := httpx.ParsePathParamInt64(c, "legalEntityId")
		if !ok {
			return
		}
		list, err := a.User.ListEntityStaff(c.Request.Context(), entityID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// orgCreateEntityStaffHandler POST /legal-entities/{id}/staff:录入企业员工登录信息。
func orgCreateEntityStaffHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		entityID, ok := httpx.ParsePathParamInt64(c, "legalEntityId")
		if !ok {
			return
		}
		var req entityStaffCreateReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.Username, "username", 64),
				httpx.RequireString(req.Password, "password", 128),
				httpx.RequireString(req.RealName, "realName", 64),
				httpx.RequireString(req.RoleCode, "roleCode", 32),
			)
		}) {
			return
		}
		id, err := a.User.CreateEntityStaff(c.Request.Context(), entityID, user.AccountInput{
			Username: req.Username, Password: req.Password, RealName: req.RealName,
			Phone: req.Phone, StaffNo: req.StaffNo, RoleCode: req.RoleCode,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "权限变更", "account", strconv.FormatInt(id, 10), gin.H{
			"op": "entity-staff-create", "legalEntityId": entityID,
			"username": req.Username, "staffNo": req.StaffNo, "roleCode": req.RoleCode,
		})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// orgEntityStaffPasswordHandler PUT /legal-entities/{id}/staff/{accountId}/password:重置登录密码。
func orgEntityStaffPasswordHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ids, ok := parseEntityStaffPath(c)
		if !ok {
			return
		}
		var req struct {
			Password string `json:"password"`
		}
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(httpx.RequireString(req.Password, "password", 128))
		}) {
			return
		}
		if err := a.User.SetEntityStaffPassword(c.Request.Context(), ids.entityID, ids.accountID, req.Password); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "权限变更", "account", c.Param("accountId"),
			gin.H{"op": "entity-staff-reset-password", "legalEntityId": ids.entityID})
		respond(c, apitypes.CodeOK, nil)
	}
}

// orgEntityStaffStatusHandler PUT /legal-entities/{id}/staff/{accountId}/status:启用(1)/停用(0)。
func orgEntityStaffStatusHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ids, ok := parseEntityStaffPath(c)
		if !ok {
			return
		}
		var req struct {
			Status *int16 `json:"status"`
		}
		if !httpx.BindAndValidate(c, &req) || req.Status == nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := a.User.SetEntityStaffStatus(c.Request.Context(), ids.entityID, ids.accountID, *req.Status); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "account", c.Param("accountId"),
			gin.H{"op": "entity-staff-status", "legalEntityId": ids.entityID, "status": *req.Status})
		respond(c, apitypes.CodeOK, nil)
	}
}

// entityStaffPath 路径双参数(legalEntityId + accountId)。
type entityStaffPath struct {
	entityID  int64
	accountID int64
}

// parseEntityStaffPath 解析并校验企业员工路径参数。
func parseEntityStaffPath(c *gin.Context) (entityStaffPath, bool) {
	entityID, ok := httpx.ParsePathParamInt64(c, "legalEntityId")
	if !ok {
		return entityStaffPath{}, false
	}
	accountID, ok := httpx.ParsePathParamInt64(c, "accountId")
	if !ok {
		return entityStaffPath{}, false
	}
	return entityStaffPath{entityID: entityID, accountID: accountID}, true
}
