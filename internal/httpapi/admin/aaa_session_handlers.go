package adminapi

// AAA-A2 在线会话管理:分页查询(loid/nasIp/status 过滤)与强制下线,权限码沿用 menu:loaccount。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// aaaSessionsPageHandler GET /aaa/sessions:在线会话分页。
func aaaSessionsPageHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		svc, ok := a.Aaa.(aaa.SessionAdminQuery)
		if !ok {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		scope, err := aaaScope(c, a.User)
		if err != nil {
			respondErr(c, err)
			return
		}
		result, err := svc.ListSessionsPage(c.Request.Context(), aaaSessionPage(c), scope)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, result)
	}
}

// aaaSessionPage 分页参数(loid/nasIp/status 过滤)。
func aaaSessionPage(c *gin.Context) aaa.SessionPage {
	return aaa.SessionPage{
		Page: queryInt(c, "page", 1), PageSize: queryInt(c, "pageSize", 20),
		Loid: c.Query("loid"), NasIP: c.Query("nasIp"), Status: c.Query("status"),
	}
}

// aaaSessionDisconnectHandler POST /aaa/sessions/:sessionId/disconnect:强制下线
// (向会话所在 NAS 发 RFC 5176 Disconnect;NAS 不可达转 PENDING_OFFLINE 由后台重试)。
func aaaSessionDisconnectHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.SessCtl == nil {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		sessionID, err := strconv.ParseInt(c.Param("sessionId"), 10, 64)
		if err != nil || sessionID <= 0 {
			respondBadRequest(c, "invalid sessionId")
			return
		}
		if err := a.SessCtl.ForceOfflineSessionByID(c.Request.Context(), sessionID, aaa.CloseReasonCoA); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "aaa_session", c.Param("sessionId"), map[string]any{"op": "force_offline"})
		respond(c, apitypes.CodeOK, gin.H{"sessionId": sessionID, "action": "disconnect"})
	}
}
