package app

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/pkg/audit"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

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

// claimsAccountID 取当前请求账号 id(未认证返回 0)。
func claimsAccountID(c *gin.Context) int64 {
	if v, ok := c.Get(middleware.CtxClaims); ok {
		if claims, ok := v.(*auth.Claims); ok {
			return claims.AccountID
		}
	}
	return 0
}
