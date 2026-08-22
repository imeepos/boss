package userapi

// 门户 Auth 域公开端点 handler 实现(从 auth.go 抽出,registerPortalAuthRoutes 只剩扁平路由表)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/portal"
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
		if ok := portalLoginVerifyPre(c, a, req.Phone, req.Mode, req.SmsCode, req.Password); !ok {
			return
		}
		if req.Mode != "sms" {
			if ok := portalLoginVerifyPassword(c, a, req.Phone, req.Password); !ok {
				return
			}
		}
		portalLoginRespond(c, mgr, a, req.Phone)
	}
}

// portalLoginVerifyPre 短信码分支预检:缺码 400,验证码错或未通过 401。
func portalLoginVerifyPre(c *gin.Context, a *app.Application, phone, mode, smsCode, password string) bool {
	if mode == "sms" {
		if smsCode == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return false
		}
		ok, err := a.Portal.ConsumeSms(c.Request.Context(), phone, "login", smsCode)
		if err != nil {
			respondErr(c, err)
			return false
		}
		if !ok {
			respond(c, apitypes.CodeUnauthorized, nil)
			return false
		}
	} else if password == "" {
		respond(c, apitypes.CodeInvalidParam, nil)
		return false
	}
	return true
}

// portalLoginVerifyPassword 密码分支验证:错则 401。
func portalLoginVerifyPassword(c *gin.Context, a *app.Application, phone, password string) bool {
	ok, err := a.Portal.VerifyPassword(c.Request.Context(), phone, password)
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
// 可选 inviteCode=邀请人手机号:配置了奖励券模板时向邀请人发一张券(尽力而为,不阻塞注册)。
func portalRegisterHandler(a *app.Application, mgr *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req portalRegisterReq
		if !httpx.BindBody(c, &req) {
			return
		}
		acc, ok := portalRegisterAccount(c, a, req)
		if !ok {
			return
		}
		portalIssueInviteReward(c, a, req.InviteCode)
		token, err := signCustomerToken(mgr, acc.CustomerID, req.Phone)
		if err != nil {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"token": token, "customerId": acc.CustomerID})
	}
}

// portalIssueInviteReward 邀请奖励:inviteCode(邀请人手机号)命中且后台配置了
// 奖励券模板时,按模板向邀请人发一张 INVITE 来源券。失败静默(注册主流程优先)。
func portalIssueInviteReward(c *gin.Context, a *app.Application, inviteCode string) {
	if inviteCode == "" || a.Promotion == nil || a.UserData == nil {
		return
	}
	cfg, err := a.UserData.GetInviteConfig(c.Request.Context())
	if err != nil || len(cfg) == 0 {
		return
	}
	tplID, _ := cfg[0]["rewardTemplateId"].(int64)
	if tplID == 0 {
		return
	}
	inviters, err := a.Customer.List(c.Request.Context(), customer.CustomerQuery{Phone: inviteCode})
	if err != nil || len(inviters) == 0 {
		return
	}
	_, _ = a.Promotion.IssueToCustomer(c.Request.Context(), tplID, inviters[0].ID, "INVITE")
}

// portalRegisterCustomerID 解析注册绑定客户ID:同名客户复用,否则取 NextSyntheticCustomerID。
func portalRegisterCustomerID(c *gin.Context, a *app.Application, phone string) (int64, error) {
	customerID, err := a.Portal.NextSyntheticCustomerID(c.Request.Context())
	if err != nil {
		return 0, err
	}
	if list, err := a.Customer.List(c.Request.Context(), customer.CustomerQuery{Phone: phone}); err == nil && len(list) > 0 {
		customerID = list[0].ID
	}
	return customerID, nil
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
		if !portalResetConsumeSms(c, a, req.Phone, req.SmsCode) {
			return
		}
		if !portalResetUpsertPassword(c, a, req.Phone, req.NewPassword) {
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
// portalRegisterReq 注册请求体。
type portalRegisterReq struct {
	Phone      string `json:"phone" binding:"required"`
	SmsCode    string `json:"smsCode" binding:"required"`
	Password   string `json:"password" binding:"required,min=10"`
	InviteCode string `json:"inviteCode"` // 邀请人手机号,可空
}

// portalRegisterAccount 注册三步:验证码核销 → 客户建档 → 账号 upsert;
// 失败已回写响应。
func portalRegisterAccount(c *gin.Context, a *app.Application, req portalRegisterReq) (*portal.Account, bool) {
	ok, err := a.Portal.ConsumeSms(c.Request.Context(), req.Phone, "register", req.SmsCode)
	if err != nil {
		respondErr(c, err)
		return nil, false
	}
	if !ok {
		respond(c, apitypes.CodeUnauthorized, nil)
		return nil, false
	}
	customerID, err := portalRegisterCustomerID(c, a, req.Phone)
	if err != nil {
		respondErr(c, err)
		return nil, false
	}
	acc, err := a.Portal.UpsertAccount(c.Request.Context(), req.Phone, req.Password, customerID)
	if err != nil {
		respondErr(c, err)
		return nil, false
	}
	return acc, true
}

// portalResetConsumeSms 重置场景验证码核销;失败已回写响应。
func portalResetConsumeSms(c *gin.Context, a *app.Application, phone, smsCode string) bool {
	ok, err := a.Portal.ConsumeSms(c.Request.Context(), phone, "reset", smsCode)
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

// portalResetUpsertPassword 按手机号回写新口令;失败已回写响应。
func portalResetUpsertPassword(c *gin.Context, a *app.Application, phone, newPassword string) bool {
	acc, err := a.Portal.AccountByPhone(c.Request.Context(), phone)
	if err != nil {
		respondErr(c, err)
		return false
	}
	if _, err := a.Portal.UpsertAccount(c.Request.Context(), phone, newPassword, acc.CustomerID); err != nil {
		respondErr(c, err)
		return false
	}
	return true
}

// portalLoginRespond 取账号 + 签发客户 token 并回执;失败已回写响应。
func portalLoginRespond(c *gin.Context, mgr *auth.Manager, a *app.Application, phone string) {
	acc, err := a.Portal.AccountByPhone(c.Request.Context(), phone)
	if err != nil {
		respondErr(c, err)
		return
	}
	token, err := signCustomerToken(mgr, acc.CustomerID, phone)
	if err != nil {
		respond(c, apitypes.CodeInternal, nil)
		return
	}
	respond(c, apitypes.CodeOK, gin.H{"token": token, "customerId": acc.CustomerID})
}
