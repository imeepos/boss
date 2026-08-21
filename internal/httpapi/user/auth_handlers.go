package userapi

// 门户 Auth 域公开端点 handler 实现(从 auth.go 抽出,registerPortalAuthRoutes 只剩扁平路由表)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalLoginHandler POST /auth/login:客户短信码/密码登录。
func portalLoginHandler(a *app.Application, mgr *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Phone    string `json:"phone" binding:"required"`
			Mode     string `json:"mode" binding:"omitempty,oneof=sms password"`
			SmsCode  string `json:"smsCode"`
			Password string `json:"password"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		if req.Mode == "sms" {
			if req.SmsCode == "" {
				respond(c, apitypes.CodeInvalidParam, nil)
				return
			}
			ok, err := a.Portal.ConsumeSms(c.Request.Context(), req.Phone, "login", req.SmsCode)
			if err != nil {
				respondErr(c, err)
				return
			}
			if !ok {
				respond(c, apitypes.CodeUnauthorized, nil)
				return
			}
		} else if req.Password == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		acc, err := a.Portal.AccountByPhone(c.Request.Context(), req.Phone)
		if err != nil {
			respondErr(c, err)
			return
		}
		if req.Mode != "sms" {
			ok, err := a.Portal.VerifyPassword(c.Request.Context(), req.Phone, req.Password)
			if err != nil {
				respondErr(c, err)
				return
			}
			if !ok {
				respond(c, apitypes.CodeUnauthorized, nil)
				return
			}
		}
		token, err := signCustomerToken(mgr, acc.CustomerID, req.Phone)
		if err != nil {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"token": token, "customerId": acc.CustomerID})
	}
}

// portalIssueSmsHandler POST /auth/sms-code:签发登录/注册/找回密码短信码。
func portalIssueSmsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Phone string `json:"phone" binding:"required"`
			Scene string `json:"scene" binding:"required,oneof=login register reset"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		if err := a.Portal.IssueSms(c.Request.Context(), req.Phone, req.Scene); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalRegisterHandler POST /auth/register:客户注册(短信码校验 + 发 token)。
func portalRegisterHandler(a *app.Application, mgr *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Phone    string `json:"phone" binding:"required"`
			SmsCode  string `json:"smsCode" binding:"required"`
			Password string `json:"password" binding:"required,min=10"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		ok, err := a.Portal.ConsumeSms(c.Request.Context(), req.Phone, "register", req.SmsCode)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !ok {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		// 契约要求注册即返回 token+customerId;已有同名手机号客户则直接绑定,否则发隔离空间合成 ID。
		customerID, err := a.Portal.NextSyntheticCustomerID(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		if list, err := a.Customer.List(c.Request.Context(), customer.CustomerQuery{Phone: req.Phone}); err == nil && len(list) > 0 {
			customerID = list[0].ID
		}
		acc, err := a.Portal.UpsertAccount(c.Request.Context(), req.Phone, req.Password, customerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		token, err := signCustomerToken(mgr, acc.CustomerID, req.Phone)
		if err != nil {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"token": token, "customerId": acc.CustomerID})
	}
}

// portalResetPasswordHandler POST /auth/reset-password:短信码重置密码。
func portalResetPasswordHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Phone       string `json:"phone" binding:"required"`
			SmsCode     string `json:"smsCode" binding:"required"`
			NewPassword string `json:"newPassword" binding:"required,min=10"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		ok, err := a.Portal.ConsumeSms(c.Request.Context(), req.Phone, "reset", req.SmsCode)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !ok {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		acc, err := a.Portal.AccountByPhone(c.Request.Context(), req.Phone)
		if err != nil {
			respondErr(c, err)
			return
		}
		if _, err := a.Portal.UpsertAccount(c.Request.Context(), req.Phone, req.NewPassword, acc.CustomerID); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}