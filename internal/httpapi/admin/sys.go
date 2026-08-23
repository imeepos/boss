package adminapi

// sys 横切路由:审计日志查询与业务参数热更(均 SYS 域,menu:audit/menu:params 门禁)。
// storage-config handler 见 storageconfig.go;此处仅 sys 域 handler 实现。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/audit"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerSysRoutes 注册 sys 横切路由。
func registerSysRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/audit-logs", requirePerm(a.User, "menu:audit"), sysAuditLogsHandler(a))
	g.GET("/params", requirePerm(a.User, "menu:params"), sysListParamsHandler(a))
	g.GET("/import-tasks", requirePerm(a.User, "menu:importer"), sysListImportTasksHandler(a))
	g.PUT("/params/:key", requirePerm(a.User, "menu:params"), sysUpdateParamHandler(a))

	g.GET("/storage-config", requirePerm(a.User, "menu:params"), adminStorageConfigGet(a))
	g.PUT("/storage-config", requirePerm(a.User, "menu:params"), adminStorageConfigPut(a))
	g.POST("/storage-config/rotate-secret", requirePerm(a.User, "menu:params"), adminStorageConfigRotateSecret(a))
}

// sysAuditLogsHandler GET /audit-logs:审计日志(按人/类型/目标类型过滤 + 分页,sys.yaml listAuditLogs)。
func sysAuditLogsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// sysListParamsHandler GET /params:业务参数全量清单(sys.yaml listParams)。
func sysListParamsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.User.ListParams(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// sysListImportTasksHandler GET /import-tasks:导入任务记录(menu:importer)。
func sysListImportTasksHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.User.ListImportTasks(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// sysUpdateParamHandler PUT /params/{key}:业务参数单项热更(sys.yaml updateParam)。
func sysUpdateParamHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		if key == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "key is required"})
			return
		}
		var req struct {
			Value string `json:"value" binding:"required"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		var accountID int64 = httpx.ClaimsAccountID(c)
		if err := a.User.UpdateParam(c.Request.Context(), key, req.Value, accountID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "biz_param", key, map[string]any{"value": req.Value})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
