package adminapi

// 崩溃日志管理端查看:GET /crash-logs,仅 sysadmin(菜单 menu:crash_logs)。
// 入库链路见 internal/domain/crash;App 端 POST /api/worker/v1/client/crash
// 留痕,管理端负责消费/排查展示。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	crashdomain "github.com/ymm-001/boss/internal/domain/crash"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func registerCrashLogRoutes(g *gin.RouterGroup, a *app.Application) {
	p := requirePerm(a.User, "menu:crash_logs")
	g.GET("/crash-logs", p, func(c *gin.Context) {
		if a.CrashLogs == nil {
			respond(c, apitypes.CodeOK, []crashdomain.Log{})
			return
		}
		limit := 50
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		logs, err := a.CrashLogs.ListRecent(c.Request.Context(), limit)
		if err != nil {
			respondErr(c, err)
			return
		}
		if logs == nil {
			logs = []crashdomain.Log{}
		}
		respond(c, apitypes.CodeOK, logs)
	})
}
