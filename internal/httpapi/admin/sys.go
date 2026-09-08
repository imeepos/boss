package adminapi

// sys 横切路由:审计日志查询与业务参数热更(均 SYS 域,menu:audit/menu:params 门禁)。
// storage-config handler 见 storageconfig.go;此处仅 sys 域 handler 实现。

import (
	"regexp"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/audit"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// entityTaskKindRe 客户端登记的 kind 白名单:entity:<蛇形小写>(与前端 entities.ts kind 对齐;前端单测镜像本正则防漂移)。
var entityTaskKindRe = regexp.MustCompile(`^entity:[a-z][a-z0-9_]{0,31}$`)

// registerSysRoutes 注册 sys 横切路由。
func registerSysRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/audit-logs", requirePerm(a.User, "menu:audit"), sysAuditLogsHandler(a))
	g.GET("/params", requirePerm(a.User, "menu:params"), sysListParamsHandler(a))
	g.GET("/import-tasks", requirePerm(a.User, "menu:importer"), sysListImportTasksHandler(a))
	g.POST("/import-tasks", requirePerm(a.User, "menu:importer"), sysRecordImportTaskHandler(a))
	g.PUT("/params/:key", requirePerm(a.User, "menu:params"), sysUpdateParamHandler(a))

	g.GET("/storage-config", requirePerm(a.User, "menu:params"), adminStorageConfigGet(a))
	g.PUT("/storage-config", requirePerm(a.User, "menu:params"), adminStorageConfigPut(a))
	g.POST("/storage-config/test", requirePerm(a.User, "menu:params"), adminStorageConfigTest(a))
	g.POST("/storage-config/rotate-secret", requirePerm(a.User, "menu:params"), adminStorageConfigRotateSecret(a))

	// 运维脚本上报通道(cron/巡检/隧道自愈):account 主体 API key 调用,
	// refType 白名单防脏数据;实现见 ops_notify.go。
	g.POST("/ops/notify-emit", requirePerm(a.User, "menu:dispatch"), opsNotifyEmitHandler(a))
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
		list, err := a.User.ListImportTasks(c.Request.Context(), c.Query("kind"), c.Query("operator"), c.Query("from"), c.Query("to"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// sysRecordImportTaskHandler POST /import-tasks:客户端批量导入结果登记(menu:importer)。
// 业务批量导入为前端逐行调用各域创建端点,完成后经此落一条任务记录(kind 形如 entity:department)。
func sysRecordImportTaskHandler(a *app.Application) gin.HandlerFunc {
	type recordReq struct {
		Kind      string         `json:"kind"`
		Total     int            `json:"total"`
		Imported  int            `json:"imported"`
		Failed    int            `json:"failed"`
		Skipped   int            `json:"skipped"`
		Detail    map[string]any `json:"detail"`
		ClientKey string         `json:"clientKey"`
	}
	return func(c *gin.Context) {
		var req recordReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(httpx.RequireString(req.Kind, "kind", 64))
		}) {
			return
		}
		if len([]rune(req.ClientKey)) > 96 {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "clientKey exceeds 96 characters"})
			return
		}
		if !entityTaskKindRe.MatchString(req.Kind) || req.Total < 0 || req.Imported < 0 || req.Failed < 0 || req.Skipped < 0 || req.Imported+req.Failed+req.Skipped > req.Total {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "kind must be entity:<name>, counts must be >= 0 and not exceed total"})
			return
		}
		if err := a.User.RecordImportTask(c.Request.Context(), req.Kind, httpx.ClaimsAccountID(c), req.Total, req.Imported, req.Failed, req.Skipped, req.Detail, req.ClientKey); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
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
