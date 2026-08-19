// Package httpapi 是三端 REST 路由的组合根:按端分包(admin/user/worker),
// 各端独立前缀与鉴权链,共享件在 internal/pkg/httpx。
package httpapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/httpapi/admin"
	"github.com/ymm-001/boss/internal/httpapi/user"
	"github.com/ymm-001/boss/internal/httpapi/worker"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// RegisterRoutes 在 gin engine 上注册三端路由;mgr 为 JWT 单事实源签发器(D1)。
func RegisterRoutes(r *gin.Engine, a *app.Application, mgr *auth.Manager) {
	adminapi.Register(r, a, mgr)
	userapi.Register(r, a, mgr)
	workerapi.Register(r, a, mgr)
}
