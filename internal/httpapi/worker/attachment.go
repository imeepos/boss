package workerapi

// 附件上传(师傅端):师傅登录后上传,上传者身份固定 worker + workerId。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/attachment"
	"github.com/ymm-001/boss/internal/pkg/httpx"
)

// registerWorkerPortalAttachmentRoutes 附件域路由(wauth 组,师傅身份强制)。
func registerWorkerPortalAttachmentRoutes(g *gin.RouterGroup, a *app.Application) {
	g.POST("/attachments/upload", httpx.AttachmentUpload(a.Attachment, workerAttachmentUploader))
}

// workerAttachmentUploader 身份解析:当前师傅(workerJWT 注入的 ctxPortalWorkerID)。
func workerAttachmentUploader(c *gin.Context) (string, int64, bool) {
	id, _ := portalWorker(c)
	if id <= 0 {
		return "", 0, false
	}
	return attachment.UploaderWorker, id, true
}
