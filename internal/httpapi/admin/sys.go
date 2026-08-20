package adminapi

// sys 横切路由:审计日志查询与业务参数热更(均 SYS 域,menu:audit/menu:params 门禁)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/audit"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func registerSysRoutes(g *gin.RouterGroup, a *app.Application) {
	// 审计日志:按人/类型/目标类型过滤 + 分页(sys.yaml listAuditLogs)。
	g.GET("/audit-logs", requirePerm(a.User, "menu:audit"), func(c *gin.Context) {
		if a.Audit == nil {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		q := audit.Query{
			AccountID:  queryInt64(c, "accountId"),
			Action:     c.Query("action"),
			TargetType: c.Query("targetType"),
			Limit:      int(queryInt64(c, "limit")),
			Offset:     int(queryInt64(c, "offset")),
		}
		if q.Limit <= 0 || q.Limit > 500 {
			q.Limit = 100
		}
		list, err := a.Audit.List(c.Request.Context(), q)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	// 业务参数:全量清单 + 单项热更(sys.yaml listParams/updateParam)。
	g.GET("/params", requirePerm(a.User, "menu:params"), func(c *gin.Context) {
		list, err := a.User.ListParams(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	// 导入任务记录:数据导入中心历史清单(menu:importer)。
	g.GET("/import-tasks", requirePerm(a.User, "menu:importer"), func(c *gin.Context) {
		list, err := a.User.ListImportTasks(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.PUT("/params/:key", requirePerm(a.User, "menu:params"), func(c *gin.Context) {
		var req struct {
			Value string `json:"value" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		var accountID int64 = httpx.ClaimsAccountID(c)
		if err := a.User.UpdateParam(c.Request.Context(), c.Param("key"), req.Value, accountID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "biz_param", c.Param("key"), map[string]any{"value": req.Value})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	g.GET("/storage-config", requirePerm(a.User, "menu:params"), adminStorageConfigGet(a))
	g.PUT("/storage-config", requirePerm(a.User, "menu:params"), adminStorageConfigPut(a))
}
