package adminapi

// 数据备份迁移路由(SYS 域运维工具,迁移 000095;menu:backup 门禁)。
// 契约:GET/POST /backup/*,字段口径 docs/contract/fields.md 1.5.6。
// 全部 handler 实现见 backup_handlers.go;此处只保留扁平路由表与共享工具。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

// backupOperatorID 从 JWT claims 取操作账号;API key 主体无账号概念,拒绝。
func backupOperatorID(c *gin.Context) (int64, bool) {
	claims, ok := c.Get(middleware.CtxClaims)
	if !ok {
		return 0, false
	}
	cl, ok := claims.(*auth.Claims)
	if !ok || cl.AccountID <= 0 {
		return 0, false
	}
	return cl.AccountID, true
}

// registerBackupRoutes 注册备份迁移路由;服务未装配(归档目录建失败)时统一回内部错误。
func registerBackupRoutes(g *gin.RouterGroup, a *app.Application) {
	if a.Backup == nil {
		g.GET("/backup/tables", backupTablesUnavailableHandler(a))
		return
	}
	perm := requirePerm(a.User, "menu:backup")

	g.GET("/backup/tables", perm, backupListTablesHandler(a))
	g.GET("/backup/jobs", perm, backupListJobsHandler(a))
	g.POST("/backup/jobs", perm, backupCreateJobHandler(a))
	g.POST("/backup/restore", perm, backupRestoreHandler(a))

	registerBackupJobItemRoutes(g, a, perm)
}

// registerBackupJobItemRoutes 任务级路由:详情/删除/下载。
func registerBackupJobItemRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.GET("/backup/jobs/:id", perm, backupGetJobHandler(a))
	g.DELETE("/backup/jobs/:id", perm, backupDeleteJobHandler(a))
	g.GET("/backup/jobs/:id/file", perm, backupDownloadJobHandler(a))
}