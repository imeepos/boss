package app

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/audit"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// respond 统一响应 envelope:{code,msg,data};跨进程错误码对齐 pkg/apitypes(D3)。
func respond(c *gin.Context, code apitypes.Code, data any) {
	c.JSON(http.StatusOK, gin.H{"code": code, "msg": code.Message(), "data": data})
}

// respondErr 领域错误 → 统一错误码。未知错误一律 500。
func respondErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, user.ErrUnauthorized):
		respond(c, apitypes.CodeUnauthorized, nil)
	case errors.Is(err, user.ErrNotFound),
		errors.Is(err, resource.ErrNotFound),
		errors.Is(err, asset.ErrNotFound),
		errors.Is(err, customer.ErrCustomerNotFound),
		errors.Is(err, order.ErrOrderNotFound):
		respond(c, apitypes.CodeNotFound, nil)
	case errors.Is(err, resource.ErrIllegalTransition),
		errors.Is(err, resource.ErrPortNotAvailable),
		errors.Is(err, order.ErrIllegalTransition):
		respond(c, apitypes.CodeInvalidParam, nil)
	default:
		respond(c, apitypes.CodeInternal, nil)
	}
}

// recordAudit 记录关键操作审计(异步、尽力而为);未装配审计 writer 时静默跳过。
func (a *Application) recordAudit(c *gin.Context, action, targetType, targetID string, detail map[string]any) {
	if a == nil || a.Audit == nil {
		return
	}
	var accountID int64
	if v, ok := c.Get(middleware.CtxClaims); ok {
		if claims, ok := v.(*auth.Claims); ok {
			accountID = claims.AccountID
		}
	}
	_ = a.Audit.Write(c.Request.Context(), audit.Event{
		AccountID: accountID, Action: action, TargetType: targetType, TargetID: targetID,
		Detail: detail, IP: c.ClientIP(),
	})
}

// loginReq 登录请求体(对齐 api/openapi/admin/auth.yaml)。
type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRoutes 在 gin engine 上注册业务路由;mgr 为 JWT 单事实源签发器(D1)。
func RegisterRoutes(r *gin.Engine, a *Application, mgr *auth.Manager) {
	api := r.Group("/api/v1")
	api.POST("/auth/login", func(c *gin.Context) {
		var req loginReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		res, err := a.User.Login(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			respondErr(c, err)
			return
		}
		token, err := mgr.Sign(res.AccountID, res.Username, res.RoleCode)
		if err != nil {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"token":     token,
			"accountId": res.AccountID,
			"realName":  res.RealName,
			"roleName":  res.RoleName,
		})
	})

	// 需要鉴权的路由组:登录后经 JWT 认证;RBAC 逐接口注入 permCode。
	authed := api.Group("")
	authed.Use(middleware.Authn(mgr))

	authed.GET("/auth/me", func(c *gin.Context) {
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		p, err := a.User.GetProfile(c.Request.Context(), claims.AccountID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, p)
	})

	registerOrgRoutes(authed, a)
	registerOrderRoutes(authed, a)
	registerBillingRoutes(authed, a)
	registerCustomerRoutes(authed, a)
	registerResourceRoutes(authed, a)
	registerAssetRoutes(authed, a)
	registerAaaRoutes(authed, a)
	registerDeviceRoutes(authed, a)
	registerProvisionRoutes(authed, a)
	registerQuadlinkRoutes(authed, a)
	registerWorkerRoutes(authed, a)
}
