package adminapi

// 附件上传(admin 端):登录账号或 API key 主体(account/worker/customer)均可上传,
// 上传者身份随主体类型落 attachments 表。

import (
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

// adminAttachmentList 列出指定上传者的附件;缺省按当前登录身份。
func adminAttachmentList(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ut, uid, ok := adminAttachmentUploader(c)
		if !ok {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		list, err := a.Attachment.St.ListByUploader(c.Request.Context(), ut, uid, 50)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}
