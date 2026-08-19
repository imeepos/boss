package userapi

// 用户端门户 Profile 域:实名认证(/auth/verify) + 我的/账号安全/通知订阅/语言。

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/pkg/httpx"
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
func portalVerifyStatus(a *app.Application) gin.HandlerFunc {
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
func portalVerifySubmit(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			IdType string `json:"idType"`
			Name   string `json:"name" binding:"required"`
			IdNo   string `json:"idNo" binding:"required"`
		}
		if !httpx.BindBody(c, &req) {
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
func portalProfile(a *app.Application) gin.HandlerFunc {
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
			"plan": portalProfilePlan(a, c, v.ID),
		})
	}
}

// portalProfilePlan /profile 的 plan 聚合:当前套餐(user_plans + 产品价)
// + 本月账单(最近未缴账单;无未缴返回 0 与空串,端上显示"—")。
func portalProfilePlan(a *app.Application, c *gin.Context, cid int64) gin.H {
	plan := gin.H{"planId": "", "name": "", "monthlyFee": 0, "contractEnd": "",
		"status": "NONE", "installAddress": "", "currentBillAmount": 0, "currentBillDue": ""}
	if row, ok := portalCurrentPlanRow(a, c, cid); ok {
		pid := toInt64(row["productId"])
		plan["planId"] = fmt.Sprintf("%d", pid)
		plan["name"] = toStr(row["planName"])
		plan["status"] = toStr(row["status"])
		if p := portalProductByID(a, c, pid); p != nil {
			plan["monthlyFee"] = p.MonthlyFee
		}
		plan["installAddress"] = portalDefaultAddress(a, c, cid)
	}
	if bills, err := a.Billing.ListBills(c.Request.Context(), cid); err == nil {
		for _, b := range bills {
			if b.Status != "PAID" && b.Period >= toStr(plan["currentBillDue"]) {
				plan["currentBillAmount"] = b.Amount
				plan["currentBillDue"] = b.Period
			}
		}
	}
	return plan
}

// portalCurrentPlanRow 客户当前套餐行(user_plans 最近一条)。
func portalCurrentPlanRow(a *app.Application, c *gin.Context, cid int64) (map[string]any, bool) {
	if a.UserData == nil {
		return nil, false
	}
	rows, err := a.UserData.ListUserPlans(c.Request.Context())
	if err != nil {
		return nil, false
	}
	for _, r := range rows {
		if toInt64(r["customerId"]) == cid {
			return r, true
		}
	}
	return nil, false
}

// portalDefaultAddress 默认地址明细(user_addresses is_default)。
func portalDefaultAddress(a *app.Application, c *gin.Context, cid int64) string {
	if a.UserData == nil {
		return ""
	}
	rows, err := a.UserData.ListUserAddresses(c.Request.Context())
	if err != nil {
		return ""
	}
	for _, r := range rows {
		if toInt64(r["customerId"]) == cid && toBool(r["isDefault"]) {
			return toStr(r["detail"])
		}
	}
	return ""
}

// portalProductByID 产品目录按 ID 寻址(目录未接入返回 nil)。
func portalProductByID(a *app.Application, c *gin.Context, id int64) *customer.ProductOffer {
	if a.Product == nil {
		return nil
	}
	list, err := a.Product.ListProducts(c.Request.Context(), 0)
	if err != nil {
		return nil
	}
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}

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
		respond(c, apitypes.CodeOK, gin.H{
			"realNameStatus": v.RealNameStatus, "nameMasked": portalMaskName(v.Name),
			"idNoMasked": portalMaskIDNo(v.IdNo), "passwordUpdatedAt": pwdAt,
			"phoneMasked": portalMaskPhone(v.Phone),
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

func portalGetNotify(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		p, err := a.Portal.GetPrefs(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, p.Notify)
	}
}

func portalPutNotify(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var body gin.H
		if !httpx.BindBody(c, &body) {
			return
		}
		if err := a.Portal.SavePrefs(c.Request.Context(), cid, body, ""); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func portalPutLanguage(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			Language string `json:"language" binding:"required,oneof=zh en fil"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		if err := a.Portal.SavePrefs(c.Request.Context(), cid, nil, req.Language); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
