package adminapi

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/attachment"
	"github.com/ymm-001/boss/internal/domain/cms"
)

// sitePublicImgHandler 官网匿名正文图片流:仅吐"该 slug 已发布正文确实引用"的图片附件。
// 与封面端点同思路:不暴露 /attachments/:id 通用读,防枚举他人附件;
// 非 PUBLISHED/未被引用/非 image/* 一律 404,8MB 上限。
func sitePublicImgHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.CMS == nil || a.Attachment == nil {
			c.Status(404)
			return
		}
		id, err := strconv.ParseInt(c.Param("attId"), 10, 64)
		if err != nil || id <= 0 {
			c.Status(404)
			return
		}
		p, err := a.CMS.GetPublishedBySlug(c.Request.Context(), c.Param("slug"), cms.NormalizeLang(c.Query("lang")))
		if err != nil {
			c.Status(404)
			return
		}
		referenced := false
		for _, ref := range cms.AttRefs(p.Content) {
			if ref == id {
				referenced = true
				break
			}
		}
		if !referenced {
			c.Status(404)
			return
		}
		at, r, err := a.Attachment.Download(c.Request.Context(), id)
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
