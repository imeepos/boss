// notify 域 admin 路由:后台提醒中心清单/未读数/已读(docs/plan/admin-notify-center.md §3)。
// 权限复用 menu:dispatch(消息中心页同门禁,不新增迁移)。
// handler 实现在 notify_handlers.go。
package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

func registerNotifyRoutes(g *gin.RouterGroup, a *app.Application) {
	authed := g.Group("", requirePerm(a.User, "menu:dispatch"))
	authed.GET("/notifications", notifyList(a))
	authed.GET("/notifications/unread-count", notifyUnreadCount(a))
	authed.POST("/notifications/read", notifyMarkRead(a))
}

func roleOf(c *gin.Context) string {
	return c.MustGet(middleware.CtxClaims).(*auth.Claims).RoleCode
}

func accountIDOf(c *gin.Context) int64 {
	return c.MustGet(middleware.CtxClaims).(*auth.Claims).AccountID
}
