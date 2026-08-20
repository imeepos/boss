package httpx

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/attachment"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// maxUploadSize 单文件上限 32MB,超限拒绝。
const maxUploadSize = 32 << 20

// AttachmentUpload 通用附件上传 handler:multipart 字段 file,登录身份由
// UploaderFunc 从各端 gin context 解析(account/worker/customer)。
// 三端(admin/user/worker)共用同一份逻辑,仅身份来源不同。
func AttachmentUpload(svc *attachment.Service, uploader func(*gin.Context) (string, int64, bool)) gin.HandlerFunc {
	return func(c *gin.Context) {
		ut, uid, ok := uploader(c)
		if !ok {
			Respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		fh, err := c.FormFile("file")
		if err != nil {
			Respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if fh.Size > maxUploadSize {
			Respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		f, err := fh.Open()
		if err != nil {
			Respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		defer f.Close()
		at, err := svc.Upload(c.Request.Context(), &attachment.Attachment{
			FileName: fh.Filename, ContentType: fh.Header.Get("Content-Type"),
			UploaderType: ut, UploaderID: uid,
		}, f, fh.Size)
		if err != nil {
			RespondErr(c, err)
			return
		}
		Respond(c, apitypes.CodeOK, at)
	}
}
