package workerapi

// 客户端崩溃日志上报(师傅端):App 启动时补传本地留痕(契约 worker/misc.yaml)。
// 尽力而为:入库失败返回 5xx,App 保留本地文件下次再传。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	crashdomain "github.com/ymm-001/boss/internal/domain/crash"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerWorkerCrashRoutes 崩溃上报路由(wauth 组,师傅鉴权)。
func registerWorkerCrashRoutes(g *gin.RouterGroup, a *app.Application) {
	g.POST("/client/crash", func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		var req struct {
			App string `json:"app"`
			Log string `json:"log" binding:"required"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if a.CrashLogs == nil {
			respond(c, apitypes.CodeOK, gin.H{"ok": false})
			return
		}
		if err := a.CrashLogs.Insert(c.Request.Context(), crashdomain.Log{
			SubjectType: "worker", SubjectID: workerID, App: req.App, Log: req.Log,
		}); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
}
