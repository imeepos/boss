package adminapi

// 附件上传(admin 端):登录账号或 API key 主体(account/worker/customer)均可上传,
// 上传者身份随主体类型落 attachments 表。

import (
	"errors"
	"io"
	"strconv"
	"strings"

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

// maxContentBytes 内容下载读侧上限(与上传 32MB 对齐)。
const maxContentBytes = 32 << 20
func registerAttachmentRoutes(g *gin.RouterGroup, a *app.Application) {
	g.POST("/attachments/upload", httpx.AttachmentUpload(a.Attachment, adminAttachmentUploader))
	g.GET("/attachments", adminAttachmentList(a))
	g.DELETE("/attachments/:id", adminAttachmentDelete(a))
	g.POST("/attachments/batch-get", adminAttachmentBatchGet(a))
	g.GET("/attachments/:id/content", adminAttachmentContent(a))
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
		f, ok2 := attachmentListFilter(c, ut, uid)
		if !ok2 {
			return
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

// adminAttachmentContent 输出附件对象字节(导入中心等消费方按文件读取)。
// 二进制流不走 envelope;40400=不存在或已删。上传时限 32MB,读侧同限防御。
func adminAttachmentContent(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		at, r, err := a.Attachment.Download(c.Request.Context(), id)
		if err != nil {
			if errors.Is(err, attachment.ErrNotFound) {
				respond(c, apitypes.CodeNotFound, nil)
				return
			}
			respondErr(c, err)
			return
		}
		defer r.Close()
		buf, err := io.ReadAll(io.LimitReader(r, maxContentBytes+1))
		if err != nil || len(buf) > maxContentBytes {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		name := strings.ReplaceAll(at.FileName, `"`, "")
		c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
		c.Data(200, at.ContentType, buf)
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

// attachmentListFilter 解析列表筛选:类型+id 成对才有效;缺省回落请求者本人;
// 只传 id 未传类型无法定位多态主体,拒绝。失败已回写响应。
func attachmentListFilter(c *gin.Context, ut string, uid int64) (attachment.ListFilter, bool) {
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
			return f, false
		}
		return f, true
	}
	if f.UploaderID > 0 {
		respond(c, apitypes.CodeInvalidParam, nil)
		return f, false
	}
	f.UploaderType, f.UploaderID = ut, uid
	return f, true
}
