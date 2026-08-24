package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/cms"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// sitePublicView 官网匿名读投影:不含 version/authorName/管理状态。
func sitePublicView(p cms.Post) gin.H {
	return gin.H{"slug": p.Slug, "title": p.Title, "category": p.Category,
		"summary": p.Summary, "coverAttachmentId": p.CoverAttachment,
		"content": p.Content, "publishedAt": p.PublishedAt}
}

func sitePublicListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.CMS == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		items, err := a.CMS.ListPublished(c.Request.Context(), c.Query("category"), siteLimit(c))
		if err != nil {
			respondErr(c, err)
			return
		}
		out := make([]gin.H, 0, len(items))
		for _, p := range items {
			out = append(out, sitePublicView(p))
		}
		respond(c, apitypes.CodeOK, gin.H{"items": out})
	}
}

func sitePublicDetailHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.CMS == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		p, err := a.CMS.GetPublishedBySlug(c.Request.Context(), c.Param("slug"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, sitePublicView(*p))
	}
}

func siteListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.CMS == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		items, err := a.CMS.ListPosts(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

func siteCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var p cms.Post
		if !httpx.BindBody(c, &p) {
			return
		}
		id, err := a.CMS.CreatePost(c.Request.Context(), p)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

func siteUpdateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "postId")
		if !ok {
			return
		}
		var p cms.Post
		if !httpx.BindBody(c, &p) {
			return
		}
		p.ID = id
		if err := a.CMS.UpdatePost(c.Request.Context(), p); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func siteDeleteHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "postId")
		if !ok {
			return
		}
		if err := a.CMS.DeletePost(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
