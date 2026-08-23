package adminapi

import (
	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/cs"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func knowledgeListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.Knowledge == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		items, err := a.Knowledge.ListArticles(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

func knowledgeCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var article cs.KnowledgeArticle
		if !httpx.BindBody(c, &article) {
			return
		}
		id, err := a.Knowledge.CreateArticle(c.Request.Context(), article)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

func knowledgeUpdateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "articleId")
		if !ok {
			return
		}
		var article cs.KnowledgeArticle
		if !httpx.BindBody(c, &article) {
			return
		}
		article.ID = id
		if err := a.Knowledge.UpdateArticle(c.Request.Context(), article); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func knowledgeDeleteHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "articleId")
		if !ok {
			return
		}
		if err := a.Knowledge.DeleteArticle(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
