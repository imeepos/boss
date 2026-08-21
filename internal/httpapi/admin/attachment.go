package adminapi

// 附件上传(admin 端):登录账号或 API key 主体(account/worker/customer)均可上传,
// 上传者身份随主体类型落 attachments 表。

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/domain/attachment"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerAttachmentRoutes 附件域路由(authed 组,须登录)。
func registerAttachmentRoutes(g *gin.RouterGroup, a *app.Application) {
	g.POST("/attachments/upload", httpx.AttachmentUpload(a.Attachment, adminAttachmentUploader))
	g.GET("/attachments", adminAttachmentList(a))
	g.DELETE("/attachments/:id", adminAttachmentDelete(a))
	g.POST("/attachments/batch-get", adminAttachmentBatchGet(a))
}

// adminAttachmentUploader 身份解析:API key 主体优先,否则 JWT 账号。
func adminAttachmentUploader(c *gin.Context) (string, int64, bool) {
	if s := middleware.SubjectFrom(c); s != nil {
		switch s.Type {
		case apikey.SubjectAccount:
			return attachment.UploaderAccount, s.Ref, true
		case apikey.SubjectWorker:
			return attachment.UploaderWorker, s.Ref, true
		case apikey.SubjectCustomer:
			return attachment.UploaderCustomer, s.Ref, true
		}
	}
	v, ok := c.Get(middleware.CtxClaims)
	if !ok {
		return "", 0, false
	}
	claims, ok := v.(*auth.Claims)
	if !ok || claims.AccountID <= 0 {
		return "", 0, false
	}
	return attachment.UploaderAccount, claims.AccountID, true
}

// adminAttachmentList 附件查询:uploaderType+uploaderId 可选,缺省按当前登录身份;
// 支持文件名关键词 + 分页。返回 {items, total}。
func adminAttachmentList(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ut, uid, ok := adminAttachmentUploader(c)
		if !ok {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		f := attachment.ListFilter{
			UploaderType: c.Query("uploaderType"),
			UploaderID:   queryInt64(c, "uploaderId"),
			Keyword:      c.Query("keyword"),
			Limit:        int(queryInt64(c, "limit")),
			Offset:       int(queryInt64(c, "offset")),
		}
		if f.UploaderType != "" {
			if !attachment.ValidUploaderType(f.UploaderType) || f.UploaderID <= 0 {
				respond(c, apitypes.CodeInvalidParam, nil)
				return
			}
		} else if f.UploaderID > 0 {
			// 只传了 id 未传类型:无法定位多态主体,拒绝。
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		} else {
			f.UploaderType, f.UploaderID = ut, uid
		}
		items, total, err := a.Attachment.St.List(c.Request.Context(), f)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items, "total": total})
	}
}

// adminAttachmentDelete 软删除附件(置 deleted_at,MinIO 对象保留)。
func adminAttachmentDelete(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.Attachment.St.Delete(c.Request.Context(), id); err != nil {
			if errors.Is(err, attachment.ErrNotFound) {
				respond(c, apitypes.CodeNotFound, nil)
				return
			}
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "attachment.delete", "attachment", strconv.FormatInt(id, 10), nil)
		respond(c, apitypes.CodeOK, nil)
	}
}

// adminAttachmentBatchGet 按 id 批量查附件(选择回显场景)。
func adminAttachmentBatchGet(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			IDs []int64 `json:"ids"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 || len(req.IDs) > 200 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		list, err := a.Attachment.St.GetByIDs(c.Request.Context(), req.IDs)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}
