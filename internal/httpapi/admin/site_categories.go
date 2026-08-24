package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/cms"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// 分类字典管理(menu:site 权限):/site-categories CRUD。
// code 被文章引用时禁删/禁改 code(域层裁定),前端据此提示。
func siteCatListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.CMS == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		items, err := a.CMS.ListCategories(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

func siteCatCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cat cms.Category
		if !httpx.BindBody(c, &cat) {
			return
		}
		id, err := a.CMS.CreateCategory(c.Request.Context(), cat)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

func siteCatUpdateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "catId")
		if !ok {
			return
		}
		var cat cms.Category
		if !httpx.BindBody(c, &cat) {
			return
		}
		cat.ID = id
		if err := a.CMS.UpdateCategory(c.Request.Context(), cat); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func siteCatDeleteHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "catId")
		if !ok {
			return
		}
		if err := a.CMS.DeleteCategory(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
