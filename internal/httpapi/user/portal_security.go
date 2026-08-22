package userapi

// 门户 Profile 安全子域:账号安全信息 / 改密 / 换绑手机号。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalSecurity GET /profile/security。
func portalSecurity(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		v, err := a.Customer.Get(c.Request.Context(), cid)
		if err != nil {
			respond(c, apitypes.CodeOK, portalSecurityFallback(c, a, cid))
			return
		}
		pwdAt := portalSecurityPasswordAt(c, a, v.Phone)
		verifyAt := portalSecurityVerifyAt(c, a, cid)
		respond(c, apitypes.CodeOK, gin.H{
			"realNameStatus": v.RealNameStatus, "nameMasked": portalMaskName(v.Name),
			"idNoMasked": portalMaskIDNo(v.IdNo), "passwordUpdatedAt": pwdAt,
			"phoneMasked": portalMaskPhone(v.Phone), "verifyAt": verifyAt,
		})
	}
}

// portalSecurityFallback 客户档案缺失时返回的安全信息(姓名/身份证均按空)。
func portalSecurityFallback(c *gin.Context, a *app.Application, cid int64) gin.H {
	phone := portalCustomerPhone(c.Request.Context(), a, cid)
	pwdAt := ""
	if acc, accountErr := a.Portal.AccountByCustomer(c.Request.Context(), cid); accountErr == nil {
		pwdAt = acc.PasswordUpdatedAt.Format(time.RFC3339)
	}
	return gin.H{
		"realNameStatus": "NONE", "nameMasked": "", "idNoMasked": "",
		"passwordUpdatedAt": pwdAt, "phoneMasked": portalMaskPhone(phone), "verifyAt": "",
	}
}

// portalSecurityPasswordAt 最近一次改密时间(失败返回空串)。
func portalSecurityPasswordAt(c *gin.Context, a *app.Application, phone string) string {
	acc, err := a.Portal.AccountByPhone(c.Request.Context(), phone)
	if err != nil {
		return ""
	}
	return acc.PasswordUpdatedAt.Format(time.RFC3339)
}

// portalSecurityVerifyAt 最近一次 PASS 核验时间;契约要求返回。
func portalSecurityVerifyAt(c *gin.Context, a *app.Application, cid int64) string {
	vs, err := a.RealName.ListVerifications(c.Request.Context(), cid)
	if err != nil {
		return ""
	}
	verifyAt := ""
	for _, ver := range vs {
		if ver.Result == "PASS" {
			if at := ver.VerifiedAt.Format(time.RFC3339); at > verifyAt {
				verifyAt = at
			}
		}
	}
	return verifyAt
}

// portalChangePassword PUT /profile/security/password:校验旧密码 → 更新(portal_accounts 落库)。
func portalChangePassword(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			OldPassword string `json:"oldPassword" binding:"required"`
			NewPassword string `json:"newPassword" binding:"required,min=10"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		acc, err := a.Portal.AccountByCustomer(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		ok, err := a.Portal.VerifyPassword(c.Request.Context(), acc.Phone, req.OldPassword)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !ok {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		if _, err := a.Portal.UpsertAccount(c.Request.Context(), acc.Phone, req.NewPassword, cid); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalChangePhone PUT /profile/security/phone:验证码换绑(新号 scene=login 码,portal_accounts 落库)。
func portalChangePhone(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			NewPhone string `json:"newPhone" binding:"required"`
			SmsCode  string `json:"smsCode" binding:"required"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		ok, err := a.Portal.ConsumeSms(c.Request.Context(), req.NewPhone, "login", req.SmsCode)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !ok {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		if err := a.Portal.RebindPhone(c.Request.Context(), cid, req.NewPhone); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
