// notify 域 admin 路由:后台提醒中心清单/未读数/已读(docs/plan/admin-notify-center.md §3)。
// 权限复用 menu:dispatch(消息中心页同门禁,不新增迁移)。
package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func registerNotifyRoutes(g *gin.RouterGroup, a *app.Application) {
	authed := g.Group("", requirePerm(a.User, "menu:dispatch"))

	authed.GET("/notifications", func(c *gin.Context) {
		f := notify.Filter{
			Category:     c.Query("category"),
			Level:        c.Query("level"),
			Unread:       c.Query("unread") == "1",
			HideResolved: c.Query("hideResolved") == "1",
			Limit:        int(queryInt64(c, "limit")),
			Offset:       int(queryInt64(c, "offset")),
		}
		items, total, err := a.Notify.List(c.Request.Context(), roleOf(c), accountIDOf(c), f)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items, "total": total})
	})

	authed.GET("/notifications/unread-count", func(c *gin.Context) {
		n, err := a.Notify.UnreadCount(c.Request.Context(), roleOf(c), accountIDOf(c))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"count": n})
	})

	authed.POST("/notifications/read", func(c *gin.Context) {
		var req struct {
			IDs []int64 `json:"ids"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "bad body"})
			return
		}
		if err := a.Notify.MarkRead(c.Request.Context(), roleOf(c), accountIDOf(c), req.IDs); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
}

func roleOf(c *gin.Context) string {
	return c.MustGet(middleware.CtxClaims).(*auth.Claims).RoleCode
}

func accountIDOf(c *gin.Context) int64 {
	return c.MustGet(middleware.CtxClaims).(*auth.Claims).AccountID
}
