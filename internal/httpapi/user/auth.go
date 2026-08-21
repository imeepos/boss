package userapi

// 用户端门户(客户门户)Auth 域:注册入口 + 客户 JWT + 短信码/注册/找回密码 + 门户状态存储。
// 契约单一事实源:api/openapi/user.yaml(+user/{auth,profile,misc,...}.yaml),前缀 /api/user/v1。
// 决策记录:docs/notes/adopted/2026-08-18-user-portal-customer-jwt.md。

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// customerRole 客户 JWT/API key 的固定角色码;与 roles 表 7 角色码之一对齐。
const customerRole = "customer"

// portalCustomerIDBase 未关联 customers 主档的自助注册账号 ID 基数(隔离命名空间,避免与真实主档冲突)。
const portalCustomerIDBase = int64(9_000_000_000)

// signCustomerToken 客户 JWT 签发:复用 auth.Manager(同密钥/TTL),
// AccountID=0(RBAC 恒拒,与 APIKeyAuth customer 主体同约定),客户身份编码于 Username "cust/<id>/<phone>"。
func signCustomerToken(m *auth.Manager, customerID int64, phone string) (string, error) {
	return m.Sign(auth.AudUser, 0, fmt.Sprintf("cust/%d/%s", customerID, phone), customerRole)
}

// customerIDFromToken 从 claims 解回客户 ID;非客户 token 返回 0。
func customerIDFromToken(claims *auth.Claims) int64 {
	if claims == nil || claims.RoleCode != customerRole || claims.AccountID != 0 {
		return 0
	}
	parts := strings.SplitN(claims.Username, "/", 3)
	if len(parts) != 3 || parts[0] != "cust" {
		return 0
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// requireCustomer 客户鉴权守卫:从 API key Subject 或客户 JWT 取客户 ID;非客户身份 401。
// 合成客户 ID 为负数(隔离空间),也是合法客户。
func requireCustomer(c *gin.Context) (int64, bool) {
	if s := middleware.SubjectFrom(c); s != nil && s.Type == customerRole {
		return s.Ref, true
	}
	v, ok := c.Get(middleware.CtxClaims)
	if !ok {
		respond(c, apitypes.CodeUnauthorized, nil)
		return 0, false
	}
	claims, _ := v.(*auth.Claims)
	id := customerIDFromToken(claims)
	if id == 0 {
		respond(c, apitypes.CodeUnauthorized, nil)
		return 0, false
	}
	return id, true
}

// ---- 注册入口 ----

// Register 用户端门户总入口:pub 公开端点 + uauth 客户鉴权组;前缀 /api/user/v1 与 admin 隔离。
func Register(r *gin.Engine, a *app.Application, mgr *auth.Manager) {
	pub := r.Group("/api/user/v1")
	registerCustomerSelfRegistration(pub, a)
	registerPortalAuthRoutes(pub, a, mgr)

	uauth := r.Group("/api/user/v1")
	uauth.Use(middleware.APIKeyAuth(a.APIKey, httpx.APIKeySubjectResolver(a)), middleware.Authn(mgr, auth.AudUser), portalCustomerOnly())
	registerPortalProfileRoutes(uauth, a)
	registerPortalMiscRoutes(uauth, a)
	registerPortalOrderRoutes(uauth, a)
	registerPortalBillingRoutes(uauth, a)
	registerPortalServiceRoutes(uauth, a)
	registerPortalAttachmentRoutes(uauth, a)
	registerStripeRoutes(pub, uauth, a)
}

// portalCustomerOnly 非客户身份(含 admin JWT)一律 401,客户视角端点不与后台 RBAC 混用。
func portalCustomerOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := requireCustomer(c); !ok {
			c.Abort()
			return
		}
		c.Next()
	}
}

// registerPortalAuthRoutes Auth 域公开端点(sms-code/register/reset-password)。
func registerPortalAuthRoutes(pub *gin.RouterGroup, a *app.Application, mgr *auth.Manager) {
	pub.POST("/auth/login", func(c *gin.Context) {
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
	})

	pub.POST("/auth/sms-code", func(c *gin.Context) {
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
	})

	pub.POST("/auth/register", func(c *gin.Context) {
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
	})

	pub.POST("/auth/reset-password", func(c *gin.Context) {
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
	})
}

// registerPortalProfileRoutes Profile 域(含 /auth/verify 实名;需客户身份,契约全局 bearerAuth)。
func registerPortalProfileRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/auth/verify", portalVerifyStatus(a))
	g.POST("/auth/verify", portalVerifySubmit(a))
	g.POST("/auth/verify/sms-code", portalVerifySmsCode(a))
	g.POST("/auth/logout", func(c *gin.Context) {
		// 无状态 JWT:服务端无需吊销,客户端删除本地 token 即完成登出。
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
	g.GET("/profile", portalProfile(a))
	g.GET("/profile/security", portalSecurity(a))
	g.PUT("/profile/security/password", portalChangePassword(a))
	g.PUT("/profile/security/phone", portalChangePhone(a))
	g.GET("/profile/notify-settings", portalGetNotify(a))
	g.PUT("/profile/notify-settings", portalPutNotify(a))
	g.PUT("/profile/language", portalPutLanguage(a))
}
