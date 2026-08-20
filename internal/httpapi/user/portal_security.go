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
			respondErr(c, err)
			return
		}
		pwdAt := ""
		if acc, err := a.Portal.AccountByPhone(c.Request.Context(), v.Phone); err == nil {
			pwdAt = acc.PasswordUpdatedAt.Format(time.RFC3339)
		}
		// verifyAt:契约要求返回;取最近一次 PASS 核验时间,未核验为空串。
		verifyAt := ""
		if vs, err := a.RealName.ListVerifications(c.Request.Context(), cid); err == nil {
			for _, ver := range vs {
				if ver.Result == "PASS" {
					if at := ver.VerifiedAt.Format(time.RFC3339); at > verifyAt {
						verifyAt = at
					}
				}
			}
		}
		respond(c, apitypes.CodeOK, gin.H{
			"realNameStatus": v.RealNameStatus, "nameMasked": portalMaskName(v.Name),
			"idNoMasked": portalMaskIDNo(v.IdNo), "passwordUpdatedAt": pwdAt,
			"phoneMasked": portalMaskPhone(v.Phone), "verifyAt": verifyAt,
		})
	}
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
