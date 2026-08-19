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
//
// 三端前缀与鉴权域映射(账号体系互相独立):
//
//	端       前缀               JWT                 API key 主体
//	admin   /api/admin/v1     aud=admin           account(全量 RBAC)/worker/customer(受限,扫码等场景)
//	user    /api/user/v1      aud=user            customer(仅)
//	worker  /api/worker/v1    issuer=boss-worker  (独立 claims,未接 API key)
func RegisterRoutes(r *gin.Engine, a *app.Application, mgr *auth.Manager) {
	adminapi.Register(r, a, mgr)
	userapi.Register(r, a, mgr)
	workerapi.Register(r, a, mgr)
}
