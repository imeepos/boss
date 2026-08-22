package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// createAPIKeyReq 创建 API key 请求体。
// subjectType ∈ {account, worker, customer};subjectRef 为对应主体表主键。
type createAPIKeyReq struct {
	SubjectType string `json:"subjectType" binding:"required"`
	SubjectRef  int64  `json:"subjectRef" binding:"required"`
	Name        string `json:"name" binding:"required"`
}

// registerAPIKeyRoutes 注册 API key 管理路由(需 sysadmin 权限)。
func registerAPIKeyRoutes(g *gin.RouterGroup, a *app.Application) {
	ak := g.Group("", requirePerm(a.User, "menu:apikey"))
	ak.GET("/api-keys", apikeyListHandler(a))
	ak.POST("/api-keys", apikeyCreateHandler(a))
	ak.DELETE("/api-keys/:id", apikeyRevokeHandler(a))
}
