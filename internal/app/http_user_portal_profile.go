package app

// 用户端门户 Profile 域:实名认证(/auth/verify) + 我的/账号安全/通知订阅/语言。

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalMaskPhone 138****1234;portalMaskName 王**;portalMaskIDNo 首3尾4。
func portalMaskPhone(s string) string {
	if len(s) < 7 {
		return s
	}
	return s[:3] + "****" + s[len(s)-4:]
}
func portalMaskName(s string) string {
	if s == "" {
		return ""
	}
	return s[:1] + "**"
}
func portalMaskIDNo(s string) string {
	if len(s) < 8 {
		return s
	}
	return s[:3] + strings.Repeat("*", len(s)-7) + s[len(s)-4:]
}

// portalVerifyStatus GET /auth/verify:实名状态 + 核验记录(RealName 域, terms.md real_name_status)。
func portalVerifyStatus(a *Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		records, _ := a.CustomerRealName.ListVerifications(c.Request.Context(), cid)
		items := make([]gin.H, 0, len(records))
		for _, r := range records {
			items = append(items, gin.H{"method": r.Method, "time": r.VerifiedAt, "result": r.Result})
		}
		status := "PENDING"
		if v, err := a.Customer.Get(c.Request.Context(), cid); err == nil {
			status = v.RealNameStatus
		}
		respond(c, apitypes.CodeOK, gin.H{"status": status, "records": items})
	}
}

// portalVerifySubmit POST /auth/verify:提交实名资料,落 PENDING 等后台核验。
func portalVerifySubmit(a *Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			IdType string `json:"idType"`
			Name   string `json:"name" binding:"required"`
			IdNo   string `json:"idNo" binding:"required"`
		}
		if !bindBody(c, &req) {
			return
		}
		_, err := a.CustomerRealName.SubmitRealName(c.Request.Context(), customer.CustomerRealNameVerification{
			CustomerID: cid, Method: "自助提交", RealName: req.Name, IDCardNo: req.IdNo,
			Result: customer.RealNamePending,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalProfile GET /profile:个人中心聚合(客户主档 + 实名脱敏)。
func portalProfile(a *Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		v, err := a.Customer.Get(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"customerId": v.ID, "name": v.Name, "phoneMasked": portalMaskPhone(v.Phone),
			"realName": gin.H{"nameMasked": portalMaskName(v.Name), "idType": v.IdType,
				"idNoMasked": portalMaskIDNo(v.IdNo), "status": v.RealNameStatus},
		})
	}
}

// portalSecurity GET /profile/security。
func portalSecurity(a *Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		v, err := a.Customer.Get(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		acc := portal.account(v.Phone)
		pwdAt := ""
		if acc != nil {
			pwdAt = acc.PasswordUpdatedAt.Format(time.RFC3339)
		}
		respond(c, apitypes.CodeOK, gin.H{
			"realNameStatus": v.RealNameStatus, "nameMasked": portalMaskName(v.Name),
			"idNoMasked": portalMaskIDNo(v.IdNo), "passwordUpdatedAt": pwdAt,
			"phoneMasked": portalMaskPhone(v.Phone),
		})
	}
}

// portalChangePassword PUT /profile/security/password:校验旧密码 → 更新(进程内,见报告 DB 商议)。
func portalChangePassword(c *gin.Context) {
	cid, _ := requireCustomer(c)
	var req struct {
		OldPassword string `json:"oldPassword" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required,min=10"`
	}
	if !bindBody(c, &req) {
		return
	}
	acc := portal.accountByCustomer(cid)
	if acc == nil || !portal.verifyPassword(acc.Phone, req.OldPassword) {
		respond(c, apitypes.CodeUnauthorized, nil)
		return
	}
	portal.upsertAccount(acc.Phone, req.NewPassword, cid)
	respond(c, apitypes.CodeOK, gin.H{"ok": true})
}

// portalChangePhone PUT /profile/security/phone:验证码换绑(新号 scene=login 码)。
func portalChangePhone(c *gin.Context) {
	cid, _ := requireCustomer(c)
	var req struct {
		NewPhone string `json:"newPhone" binding:"required"`
		SmsCode  string `json:"smsCode" binding:"required"`
	}
	if !bindBody(c, &req) {
		return
	}
	if !portal.checkSms(req.NewPhone, "login", req.SmsCode) {
		respond(c, apitypes.CodeUnauthorized, nil)
		return
	}
	portal.rebindPhone(cid, req.NewPhone)
	respond(c, apitypes.CodeOK, gin.H{"ok": true})
}

func portalGetNotify(c *gin.Context) {
	cid, _ := requireCustomer(c)
	respond(c, apitypes.CodeOK, portal.getPrefs(cid).Notify)
}

func portalPutNotify(c *gin.Context) {
	cid, _ := requireCustomer(c)
	var body gin.H
	if !bindBody(c, &body) {
		return
	}
	portal.savePrefs(cid, body, "")
	respond(c, apitypes.CodeOK, gin.H{"ok": true})
}

func portalPutLanguage(c *gin.Context) {
	cid, _ := requireCustomer(c)
	var req struct {
		Language string `json:"language" binding:"required,oneof=zh en fil"`
	}
	if !bindBody(c, &req) {
		return
	}
	portal.savePrefs(cid, nil, req.Language)
	respond(c, apitypes.CodeOK, gin.H{"ok": true})
}
