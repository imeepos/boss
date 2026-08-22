package adminapi

// Q2 P1 待办时限统计:GET /notifications/sla-stats(登录即可,与通知路由同链)。

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// slaStatsReader Notify 窄口断言(PGStore 实现,000115)。
type slaStatsReader interface {
	SLAStats(ctx context.Context, windowDays int) (*notify.SLAStats, error)
}

// notifySLAStatsHandler GET /notifications/sla-stats:窗口内带时限待办的
// 按时办结率(默认 7 天,windowDays 可调);Notify 无该能力时 501 语义。
func notifySLAStats(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		rd, ok := a.Notify.(slaStatsReader)
		if !ok || a.Notify == nil {
			respond(c, apitypes.CodeInvalidParam, gin.H{"msg": "sla stats unsupported"})
			return
		}
		days := int(queryInt64(c, "windowDays"))
		st, err := rd.SLAStats(c.Request.Context(), days)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, st)
	}
}
