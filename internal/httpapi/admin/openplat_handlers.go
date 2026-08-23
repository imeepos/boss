// openplat 域具名 handler(承接 registerOpenPlatRoutes 扁平路由表)。
package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/openplat"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// openPlatAppListHandler GET /openplat/apps:列出全部应用(不含 Secret)。
func openPlatAppListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.OpenPlat.ListApps(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// openPlatAppCreateHandler POST /openplat/apps:创建应用凭证,Secret 仅本次返回。
func openPlatAppCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req openPlatAppCreateReq
		if !httpx.BindAndValidate(c, &req, req.validate) {
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		res, err := a.OpenPlat.CreateApp(c.Request.Context(), req.Name, req.RateLimitRPM,
			req.DailyQuota, req.Sandbox, claims.AccountID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"id": res.ID, "appId": res.AppID, "name": res.Name,
			"rateLimitRpm": res.RateLimitRPM, "dailyQuota": res.DailyQuota,
			"sandbox": res.Sandbox, "secret": res.Secret,
		})
	}
}

// openPlatAppStatusHandler PUT /openplat/apps/{id}/status:启用/停用应用。
func openPlatAppStatusHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req openPlatAppStatusReq
		if !httpx.BindAndValidate(c, &req, nil) {
			return
		}
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.OpenPlat.SetAppStatus(c.Request.Context(), id, req.Status); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	}
}

// openPlatSubListHandler GET /openplat/apps/{id}/subscriptions:应用的事件订阅。
func openPlatSubListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		list, err := a.OpenPlat.ListSubscriptions(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// openPlatSubCreateHandler POST /openplat/apps/{id}/subscriptions:新增订阅。
func openPlatSubCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req openPlatSubCreateReq
		if !httpx.BindAndValidate(c, &req, req.validate) {
			return
		}
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		sub, err := a.OpenPlat.CreateSubscription(c.Request.Context(), id, req.EventType, req.EndpointURL)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, sub)
	}
}

// openPlatSubDeleteHandler DELETE /openplat/subscriptions/{id}:删除订阅。
func openPlatSubDeleteHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.OpenPlat.DeleteSubscription(c.Request.Context(), id); err != nil {
			if err == openplat.ErrNotFound {
				respondErr(c, err)
				return
			}
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	}
}
