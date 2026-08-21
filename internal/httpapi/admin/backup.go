package adminapi

// 数据备份迁移路由(SYS 域运维工具,迁移 000095;menu:backup 门禁)。
// 契约:GET/POST /backup/*,字段口径 docs/contract/fields.md 1.5.6。

import (
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/backup"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
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
		g.GET("/backup/tables", func(c *gin.Context) { respond(c, apitypes.CodeInternal, nil) })
		return
	}
	perm := requirePerm(a.User, "menu:backup")

	// 候选表清单(新建备份的表选择器数据源)。
	g.GET("/backup/tables", perm, func(c *gin.Context) {
		tables, err := a.Backup.ListTables(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": tables})
	})

	// 任务清单(kind/status 过滤 + 分页)。
	g.GET("/backup/jobs", perm, func(c *gin.Context) {
		f := backup.ListFilter{
			Kind:   c.Query("kind"),
			Status: c.Query("status"),
			Limit:  int(queryInt64(c, "pageSize")),
			Offset: int(queryInt64(c, "offset")),
		}
		items, total, err := a.Backup.List(c.Request.Context(), f)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items, "total": total})
	})

	// 新建备份任务:tables 空 = public 全表;异步执行,立即返回任务 id。
	g.POST("/backup/jobs", perm, func(c *gin.Context) {
		var req struct {
			Tables []string `json:"tables"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		op, ok := backupOperatorID(c)
		if !ok {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		id, err := a.Backup.CreateBackup(c.Request.Context(), op, req.Tables)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "backup.create", "backup_job", strconv.FormatInt(id, 10),
			map[string]any{"tables": len(req.Tables)})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	// 导入恢复:上传归档文件(.jsonl.gz),异步执行;只补不删(ON CONFLICT DO NOTHING)。
	g.POST("/backup/restore", perm, func(c *gin.Context) {
		op, ok := backupOperatorID(c)
		if !ok {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		fh, err := c.FormFile("file")
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "file is required"})
			return
		}
		id, err := saveAndRestore(c, a, op, fh)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "backup.restore", "backup_job", strconv.FormatInt(id, 10),
			map[string]any{"file": fh.Filename})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	registerBackupJobItemRoutes(g, a, perm)
}

// saveAndRestore 落盘上传文件并创建恢复任务。
func saveAndRestore(c *gin.Context, a *app.Application, op int64, fh *multipart.FileHeader) (int64, error) {
	base := filepath.Base(strings.ReplaceAll(fh.Filename, "\\", "/"))
	if base == "." || base == "/" || strings.Contains(base, "/") {
		return 0, backup.ErrInvalidInput
	}
	tmp, err := os.CreateTemp(a.Backup.Dir(), "restore-tmp-*")
	if err != nil {
		return 0, err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)
	if err := c.SaveUploadedFile(fh, tmpPath); err != nil {
		return 0, err
	}
	name := fmt.Sprintf("restore-%d-%s", time.Now().Unix(), base)
	return a.Backup.CreateRestore(c.Request.Context(), op, name, tmpPath)
}

// registerBackupJobItemRoutes 任务级路由:详情/删除/下载。
func registerBackupJobItemRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.GET("/backup/jobs/:id", perm, func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		job, err := a.Backup.Get(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, job)
	})

	// 删除任务行并清理归档文件。
	g.DELETE("/backup/jobs/:id", perm, func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.Backup.Delete(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "backup.delete", "backup_job", strconv.FormatInt(id, 10), nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	// 下载归档文件(仅 succeeded 的备份任务)。
	g.GET("/backup/jobs/:id/file", perm, func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		job, err := a.Backup.Get(c.Request.Context(), id)
		if err != nil {
			if errors.Is(err, backup.ErrNotFound) {
				respond(c, apitypes.CodeNotFound, nil)
				return
			}
			respondErr(c, err)
			return
		}
		if job.Kind != backup.KindBackup || job.Status != backup.StatusSucceeded || job.FileName == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		c.FileAttachment(filepath.Join(a.Backup.Dir(), job.FileName), job.FileName)
	})
}
