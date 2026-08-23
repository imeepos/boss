package adminapi

import (
	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/cs"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func registerCallbackRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:complaint")
	g.GET("/callbacks", perm, listCallbacksHandler(a))
	g.POST("/callbacks", perm, createCallbackHandler(a))
	g.PUT("/callbacks/:callbackId", perm, updateCallbackHandler(a))
	g.DELETE("/callbacks/:callbackId", perm, deleteCallbackHandler(a))
}

func listCallbacksHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Callbacks.ListCallbacks(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}
func createCallbackHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req cs.Callback
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		id, err := a.Callbacks.CreateCallback(c.Request.Context(), req)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}
func updateCallbackHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "callbackId")
		if !ok {
			return
		}
		var req cs.Callback
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if err := a.Callbacks.UpdateCallback(c.Request.Context(), id, req); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
func deleteCallbackHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "callbackId")
		if !ok {
			return
		}
		if err := a.Callbacks.DeleteCallback(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
