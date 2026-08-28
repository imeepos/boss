package adminapi

// API 在线文档域:把仓库内 OpenAPI 契约(api/openapi)聚合为自包含 JSON,
// 供 /base/apidocs 页 Swagger UI 渲染与试调(menu:apidocs 门禁,迁移 000166)。

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/api/openapi"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/openapidoc"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// docPortals 文档门户全集:portal 参数 → 聚合根文件。
var docPortals = map[string]string{
	"admin":  "admin.yaml",
	"user":   "user.yaml",
	"worker": "worker.yaml",
	"open":   "open.yaml",
}

// docBundler 契约树聚合器:进程级缓存,首个请求完成四端构建。
var docBundler = openapidoc.New(openapi.FS)

// registerApiDocsRoutes GET /docs/openapi?portal=admin|user|worker|open。
func registerApiDocsRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/docs/openapi", requirePerm(a.User, "menu:apidocs"), apiDocsSpecHandler)
}

// apiDocsSpecHandler 返回 portal 的聚合契约(envelope.data 为 OpenAPI 3 文档树)。
func apiDocsSpecHandler(c *gin.Context) {
	portal := c.DefaultQuery("portal", "admin")
	if _, ok := docPortals[portal]; !ok {
		httpx.RespondValidationError(c, "portal", "must be one of admin|user|worker|open")
		return
	}
	doc, err := docBundler.Doc(docPortals[portal])
	if err != nil {
		// 聚合失败属契约资产损坏,必须显式留痕而非静默 500。
		log.Printf("[apidocs] BUNDLE FAILED portal=%s err=%v", portal, err)
		respond(c, apitypes.CodeInternal, nil)
		return
	}
	// envelope 与全站一致(code=0);文档树只读共享,gin 每请求序列化。
	respond(c, apitypes.CodeOK, doc)
}
