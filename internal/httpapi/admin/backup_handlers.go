package adminapi

// 数据备份迁移路由具名 handler(承接 registerBackupRoutes 扁平路由表)。
// 表清单 / 任务 CRUD / 归档下载;导入恢复落盘工具 saveAndRestore 见 backup.go。

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
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// backupTablesUnavailableHandler 服务未装配(归档目录建失败)时的统一回退:内部错误。
func backupTablesUnavailableHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) { respond(c, apitypes.CodeInternal, nil) }
}

// backupListTablesHandler GET /backup/tables:候选表清单(新建备份的表选择器数据源)。
func backupListTablesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tables, err := a.Backup.ListTables(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": tables})
	}
}

// backupListJobsHandler GET /backup/jobs:任务清单(kind/status 过滤 + 分页)。
func backupListJobsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// backupCreateJobHandler POST /backup/jobs:新建备份任务;tables 空 = public 全表;异步执行,立即返回任务 id。
func backupCreateJobHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// backupRestoreHandler POST /backup/restore:导入恢复(.jsonl.gz 归档);异步执行,只补不删(ON CONFLICT DO NOTHING)。
func backupRestoreHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// backupGetJobHandler GET /backup/jobs/{id}:任务详情。
func backupGetJobHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// backupDeleteJobHandler DELETE /backup/jobs/{id}:删除任务行并清理归档文件。
func backupDeleteJobHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// backupDownloadJobHandler GET /backup/jobs/{id}/file:下载归档文件(仅 succeeded 的备份任务)。
func backupDownloadJobHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
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