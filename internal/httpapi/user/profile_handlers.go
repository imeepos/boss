package userapi

// 用户端门户 Profile 域:handler 实现(profile.go 仅留视图/数据辅助)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalVerifyStatus GET /auth/verify:实名状态 + 最新结论 + 核验记录(RealName 域, terms.md real_name_status)。
// latestResult/submitTime/rejectReason 供端上渲染 审核中/已认证/驳回 三态;records 含驳回原因。
func portalVerifyStatus(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		respond(c, apitypes.CodeOK, portalVerifyStatusPayload(c, a, cid))
	}
}

// portalVerifyStatusPayload 实名状态视图聚合(records + latest + 客户主档 / portal_accounts 回退)。
func portalVerifyStatusPayload(c *gin.Context, a *app.Application, cid int64) gin.H {
	records, _ := a.CustomerRealName.ListVerifications(c.Request.Context(), cid)
	items, latest := portalVerifyRecords(records)
	status := "PENDING"
	payload := gin.H{"status": status, "records": items}
	for k, v := range latest {
		payload[k] = v
	}
	// 优先从 customers 主档取,合成客户(隔离空间)回退 portal_accounts
	if v, err := a.Customer.Get(c.Request.Context(), cid); err == nil {
		status = v.RealNameStatus
		payload["nameMasked"] = portalMaskName(v.Name)
		payload["idNoMasked"] = portalMaskIDNo(v.IdNo)
		payload["phoneMasked"] = portalMaskPhone(v.Phone)
	} else {
		// 合成客户:从 portal_accounts 取手机号
		phone := portalCustomerPhone(c.Request.Context(), a, cid)
		payload["phoneMasked"] = portalMaskPhone(phone)
		payload["nameMasked"] = ""
		payload["idNoMasked"] = ""
	}
	payload["status"] = status
	return payload
}

// portalVerifyRecords 核验记录列表 + 最新一单(latest.*)。
func portalVerifyRecords(records []customer.RealNameVerification) ([]gin.H, gin.H) {
	items := make([]gin.H, 0, len(records))
	latest := gin.H{"latestResult": "", "submitTime": "", "rejectReason": ""}
	for _, r := range records {
		items = append(items, gin.H{"method": r.Method, "time": r.VerifiedAt, "result": r.Result, "reason": r.RejectReason})
		latest["latestResult"] = r.Result
		latest["submitTime"] = r.VerifiedAt
		if r.Result == customer.RealNameFail {
			latest["rejectReason"] = r.RejectReason
		}
	}
	return items, latest
}

// portalVerifySmsCode POST /auth/verify/sms-code:给当前客户绑定手机号发实名验证码(scene=verify)。
func portalVerifySmsCode(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		phone := portalCustomerPhone(c.Request.Context(), a, cid)
		if phone == "" {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		if err := a.Portal.IssueSms(c.Request.Context(), phone, "verify"); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "phoneMasked": portalMaskPhone(phone)})
	}
}

// portalVerifySubmit POST /auth/verify:短信验证码(scene=verify) + 证件附件(正/反面) + 实名资料,
// 落 PENDING;二要素通道启用时即时自动核验(PASS/FAIL),否则等后台人工核验。
func portalVerifySubmit(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			IdType        string `json:"idType"`
			Name          string `json:"name" binding:"required"`
			IdNo          string `json:"idNo" binding:"required"`
			SmsCode       string `json:"smsCode" binding:"required"`
			IdCardFrontID int64  `json:"idCardFrontId" binding:"required"`
			IdCardBackID  int64  `json:"idCardBackId" binding:"required"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		phone, ok := portalVerifySubmitPhone(c, a, cid)
		if !ok {
			return
		}
		if !portalVerifySubmitSms(c, a, phone, req.SmsCode) {
			return
		}
		if err := portalVerifySubmitRecord(c, a, cid, req); err != nil {
			respondErr(c, err)
			return
		}
		// 阿里云二要素自动核验(通道未配置时保持 PENDING 人工核验,见 app.AutoVerifyRealName)。
		result := a.AutoVerifyRealName(c.Request.Context(), cid, req.Name, req.IdNo)
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "result": result})
	}
}

// portalVerifySubmitPhone 实名绑定手机号校验(空 → 404)。
func portalVerifySubmitPhone(c *gin.Context, a *app.Application, cid int64) (string, bool) {
	phone := portalCustomerPhone(c.Request.Context(), a, cid)
	if phone == "" {
		respond(c, apitypes.CodeNotFound, nil)
		return "", false
	}
	return phone, true
}

// portalVerifySubmitSms 校验 verify 场景验证码(失败已 respond → false;成功 → true)。
func portalVerifySubmitSms(c *gin.Context, a *app.Application, phone, smsCode string) bool {
	ok, err := a.Portal.ConsumeSms(c.Request.Context(), phone, "verify", smsCode)
	if err != nil {
		respondErr(c, err)
		return false
	}
	if !ok {
		respond(c, apitypes.CodeUnauthorized, nil)
		return false
	}
	return true
}

// portalVerifySubmitRecord 落实名资料 PENDING 单。
func portalVerifySubmitRecord(c *gin.Context, a *app.Application, cid int64, req struct {
	IdType        string `json:"idType"`
	Name          string `json:"name" binding:"required"`
	IdNo          string `json:"idNo" binding:"required"`
	SmsCode       string `json:"smsCode" binding:"required"`
	IdCardFrontID int64  `json:"idCardFrontId" binding:"required"`
	IdCardBackID  int64  `json:"idCardBackId" binding:"required"`
}) error {
	_, err := a.CustomerRealName.SubmitRealName(c.Request.Context(), customer.CustomerRealNameVerification{
		CustomerID: cid, Method: "自助提交", RealName: req.Name, IDCardNo: req.IdNo,
		Result: customer.RealNamePending, IDCardFrontID: req.IdCardFrontID, IDCardBackID: req.IdCardBackID,
	})
	return err
}

// portalProfile GET /profile:个人中心聚合(客户主档 + 实名脱敏)。
// 合成客户(隔离空间 9e9 段,无 customers 主档)按 /home 口径降级返回,不再 40400。
func portalProfile(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		v, err := a.Customer.Get(c.Request.Context(), cid)
		if err != nil {
			respond(c, apitypes.CodeOK, portalProfileFallback(c, a, cid))
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

// portalProfileFallback 合成客户(无 customers 主档)的 /profile 降级视图。
func portalProfileFallback(c *gin.Context, a *app.Application, cid int64) gin.H {
	phone := portalCustomerPhone(c.Request.Context(), a, cid)
	return gin.H{
		"customerId": cid, "name": "用户", "phoneMasked": portalMaskPhone(phone),
		"realName": gin.H{"nameMasked": "", "idType": "", "idNoMasked": "", "status": "NONE"},
		"plan":     portalProfilePlan(a, c, cid),
	}
}

// portalGetNotify GET /profile/notify-settings。
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

// portalPutNotify PUT /profile/notify-settings。
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

// portalPutLanguage PUT /profile/language。
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
