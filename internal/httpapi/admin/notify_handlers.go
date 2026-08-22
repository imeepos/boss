package adminapi

// notify 域 handler 实现(notify.go 仅留路由表)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func notifyList(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func notifyUnreadCount(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		n, err := a.Notify.UnreadCount(c.Request.Context(), roleOf(c), accountIDOf(c))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"count": n})
	}
}

func notifyMarkRead(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}
