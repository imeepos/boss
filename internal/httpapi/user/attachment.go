package userapi

// 附件上传(用户端):客户登录后上传,上传者身份固定 customer + customerId。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/attachment"
	"github.com/ymm-001/boss/internal/pkg/httpx"
)

// registerPortalAttachmentRoutes 附件域路由(uauth 组,客户身份强制)。
func registerPortalAttachmentRoutes(g *gin.RouterGroup, a *app.Application) {
	g.POST("/attachments/upload", httpx.AttachmentUpload(a.Attachment, portalAttachmentUploader))
}

// portalAttachmentUploader 身份解析:当前客户(API key 客户主体或客户 JWT)。
func portalAttachmentUploader(c *gin.Context) (string, int64, bool) {
	if id, ok := requireCustomer(c); ok {
		return attachment.UploaderCustomer, id, true
	}
	return "", 0, false
}
