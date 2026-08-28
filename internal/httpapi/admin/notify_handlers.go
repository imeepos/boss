package adminapi

// notify 域 handler 实现(notify.go 仅留路由表)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/pkg/httpx"
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

// notifyResolve POST /notifications/resolve:按 refType+refID 办结待办(置 resolved+resolved_at)。
// 用于运维/管理员显式关闭已处置的 URGENT/WARN 待办;渠道/域回调自动办结走域层 Resolve 调用。
func notifyResolve(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RefType string `json:"refType"`
			RefID   string `json:"refID"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.RefType == "" || req.RefID == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "refType and refID required"})
			return
		}
		if err := a.Notify.Resolve(c.Request.Context(), req.RefType, req.RefID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "notification.resolve", "notification", req.RefType+"/"+req.RefID, nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
