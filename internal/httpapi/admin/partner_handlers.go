// 入驻域 handler 实现(路由表见 partner.go)。
package adminapi

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/partner"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// partnerSubmitHandler POST /partner/applications(公开):提交入驻申请。
func partnerSubmitHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req partnerSubmitReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		id, err := a.Partner.Submit(c.Request.Context(), partner.Application{
			CompanyName: req.CompanyName, CreditCode: req.CreditCode,
			ContactName: req.ContactName, ContactPhone: req.ContactPhone,
			Email: req.Email, BusinessDesc: req.BusinessDesc,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"applicationId": id, "status": partner.StatusPending})
	}
}

// partnerListHandler GET /partner/applications:审核队列(状态过滤,空=全部)。
func partnerListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Partner.ListApplications(c.Request.Context(), c.Query("status"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// partnerApproveHandler POST /partner/applications/{id}/approve:通过并开通企业账号。
// 初始口令仅本次响应返回,审核人负责线下传达;不入审计 detail(敏感)。
func partnerApproveHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		res, err := a.Partner.Approve(c.Request.Context(), id, claims.AccountID)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "partner_application.approve", "partner_application", c.Param("id"),
			gin.H{"legalEntityId": res.LegalEntityID, "adminAccountId": res.AdminAccountID})
		respond(c, apitypes.CodeOK, res)
	}
}

// partnerRejectHandler POST /partner/applications/{id}/reject:驳回记意见。
func partnerRejectHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var req workerReviewReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		if err := a.Partner.Reject(c.Request.Context(), id, claims.AccountID, req.Note); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "partner_application.reject", "partner_application", c.Param("id"),
			gin.H{"note": req.Note})
		respond(c, apitypes.CodeOK, gin.H{"status": partner.StatusRejected})
	}
}

// partnerProfileHandler GET /partner/me:入驻企业档案。
func partnerProfileHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		p, err := a.Partner.Profile(c.Request.Context(), claims.AccountID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, p)
	}
}

// partnerStaffListHandler GET /partner/staff:本企业员工列表。
func partnerStaffListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		list, err := a.Partner.ListStaff(c.Request.Context(), claims.AccountID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// partnerStaffCreateHandler POST /partner/staff:新建员工(partner_staff)。
func partnerStaffCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req partnerStaffCreateReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		id, err := a.Partner.CreateStaff(c.Request.Context(), claims.AccountID,
			req.Username, req.Password, req.RealName, req.Phone)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "partner_staff.create", "account", strconv.FormatInt(id, 10), gin.H{})
		respond(c, apitypes.CodeOK, gin.H{"staffId": id})
	}
}

// partnerStaffStatusHandler PUT /partner/staff/{id}/status:启用/停用员工。
func partnerStaffStatusHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var req partnerStaffStatusReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		if err := a.Partner.SetStaffStatus(c.Request.Context(), claims.AccountID, id, req.Status); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "partner_staff.status", "account", c.Param("id"),
			gin.H{"status": req.Status})
		respond(c, apitypes.CodeOK, nil)
	}
}

// partnerOrdersHandler GET /partner/orders:本企业订单(只读)。
func partnerOrdersHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		list, err := a.Partner.ListOrders(c.Request.Context(), claims.AccountID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}
