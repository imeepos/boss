package adminapi

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/cms"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// sitePublicView 官网匿名读投影:不含 version/authorName/管理状态。
// lang/categoryName(000155) 供官网按语言渲染与分类字典本地化展示。
func sitePublicView(p cms.Post, categoryName string) gin.H {
	return gin.H{"slug": p.Slug, "lang": p.Lang, "title": p.Title, "category": p.Category,
		"categoryName": categoryName,
		"summary":      p.Summary, "coverAttachmentId": p.CoverAttachment,
		"content": p.Content, "publishedAt": p.PublishedAt}
}

// catNameLookup 分类字典 code→语言展示名;字典读失败不阻断正文读取(回落空串,前端走枚举标签)。
func catNameLookup(a *app.Application, c *gin.Context, lang string) func(string) string {
	cats, err := a.CMS.ListCategories(c.Request.Context())
	if err != nil {
		return func(string) string { return "" }
	}
	m := make(map[string]*cms.Category, len(cats))
	for i := range cats {
		m[cats[i].Code] = &cats[i]
	}
	return func(code string) string {
		if cat := m[code]; cat != nil {
			return cat.NameFor(lang)
		}
		return ""
	}
}

func sitePublicListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.CMS == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		lang := cms.NormalizeLang(c.Query("lang"))
		items, err := a.CMS.ListPublished(c.Request.Context(), c.Query("category"), lang, siteLimit(c))
		if err != nil {
			respondErr(c, err)
			return
		}
		nameFor := catNameLookup(a, c, lang)
		out := make([]gin.H, 0, len(items))
		for _, p := range items {
			out = append(out, sitePublicView(p, nameFor(p.Category)))
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
		lang := cms.NormalizeLang(c.Query("lang"))
		p, err := a.CMS.GetPublishedBySlug(c.Request.Context(), c.Param("slug"), lang)
		if err != nil {
			respondErr(c, err)
			return
		}
		// 正文附件引用 att/N 重写为公开图片端点 URL(带 lang 校验同变体引用),官网渲染零鉴权直引。
		slug := p.Slug
		p.Content = cms.RewriteAttRefs(p.Content, func(id int64) string {
			return siteImgURLPrefix + slug + "/img/" + strconv.FormatInt(id, 10) + "?lang=" + lang
		})
		respond(c, apitypes.CodeOK, sitePublicView(*p, catNameLookup(a, c, lang)(p.Category)))
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
