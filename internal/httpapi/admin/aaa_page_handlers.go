package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func aaaLoAccountsPageHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		q, scope, svc, ok := aaaPageArgs(a, c)
		if !ok {
			return
		}
		result, err := svc.ListLoAccountsPage(c.Request.Context(), q, scope)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, result)
	}
}

func aaaCdrsPageHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		q, scope, svc, ok := aaaPageArgs(a, c)
		if !ok {
			return
		}
		result, err := svc.ListCdrsPage(c.Request.Context(), q, scope)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, result)
	}
}

func aaaAuthLogsPageHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		q, scope, svc, ok := aaaPageArgs(a, c)
		if !ok {
			return
		}
		result, err := svc.ListAuthLogsPage(c.Request.Context(), q, scope)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, result)
	}
}

func aaaPageArgs(a *app.Application, c *gin.Context) (aaa.AdminPage, aaa.AdminScope, aaa.AdminQueryService, bool) {
	svc, ok := a.Aaa.(aaa.AdminQueryService)
	if !ok {
		respond(c, apitypes.CodeInternal, nil)
		return aaa.AdminPage{}, aaa.AdminScope{}, nil, false
	}
	scope, err := aaaScope(c, a.User)
	if err != nil {
		respondErr(c, err)
		return aaa.AdminPage{}, aaa.AdminScope{}, nil, false
	}
	return aaaAdminPage(c), scope, svc, true
}
