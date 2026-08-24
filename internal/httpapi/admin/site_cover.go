package adminapi

import (
	"errors"
	"io"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/attachment"
)

// maxCoverBytes 封面读侧上限(图片远小于此,防滥用下载通道)。
const maxCoverBytes = 8 << 20

// sitePublicCoverHandler 官网匿名封面流:按 slug 定位已发布文章,仅吐其封面附件。
// 不暴露 /attachments/:id 通用读(防止枚举他人附件);非 PUBLISHED/无封面/非图片一律 404。
func sitePublicCoverHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.CMS == nil || a.Attachment == nil {
			c.Status(404)
			return
		}
		p, err := a.CMS.GetPublishedBySlug(c.Request.Context(), c.Param("slug"))
		if err != nil {
			c.Status(404)
			return
		}
		if p.CoverAttachment <= 0 {
			c.Status(404)
			return
		}
		at, r, err := a.Attachment.Download(c.Request.Context(), p.CoverAttachment)
		if err != nil {
			if errors.Is(err, attachment.ErrNotFound) {
				c.Status(404)
				return
			}
			c.Status(500)
			return
		}
		defer r.Close()
		if !strings.HasPrefix(at.ContentType, "image/") {
			c.Status(404)
			return
		}
		buf, err := io.ReadAll(io.LimitReader(r, maxCoverBytes+1))
		if err != nil || len(buf) > maxCoverBytes {
			c.Status(500)
			return
		}
		c.Header("Content-Disposition", "inline")
		c.Header("Cache-Control", "public, max-age=300")
		c.Data(200, at.ContentType, buf)
	}
}
