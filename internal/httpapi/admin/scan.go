package adminapi

// W5 扫码闭环路由注册:worker 扫码绑定/拆机扫码 + admin 扫码日志/四码对账/冲突处理。
// 全部 handler 实现见 scan_handlers.go。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

// ticket 扫码请求体(worker/scan.yaml、worker/asset.yaml)。
type scanBindReq struct {
	EPC     string `json:"epc" binding:"required"`
	Offline bool   `json:"offline"`
}

// respondScanErr 扫码域错误 → 统一错误码(不一致 40920 / 未预绑定 40910 / 未扫码 42200)。
// workerFromClaims 取操作师傅:API key worker 主体优先(复合场景测试),
// 否则回退 JWT claims(账号ID + 用户名)。
func workerFromClaims(c *gin.Context) (int64, string) {
	if s := middleware.SubjectFrom(c); s != nil && s.Type == "worker" {
		return s.Ref, s.Name
	}
	if v, ok := c.Get(middleware.CtxClaims); ok {
		if claims, ok := v.(*auth.Claims); ok {
			return claims.AccountID, claims.Username
		}
	}
	return 0, ""
}

// registerScanRoutes 注册 worker 扫码闭环 + admin 四码对账路由。
func registerScanRoutes(g *gin.RouterGroup, a *app.Application) {
	w := g.Group("/tickets")
	w.POST("/:ticketNo/scan-bind", scanBindHandler(a))
	w.POST("/:ticketNo/dismantle/scan", scanDismantleHandler(a))

	q := g.Group("", requirePerm(a.User, "menu:quadlink"))
	q.GET("/scan-logs", scanLogsHandler(a))
	q.GET("/quad-conflicts", quadConflictsListHandler(a))
	q.POST("/quad-conflicts/:id/resolve", quadConflictResolveHandler(a))
	q.POST("/quad-links/reconcile", quadLinksReconcileHandler(a))
	q.POST("/quad-links/purge-orphans", quadLinksPurgeOrphansHandler(a))
}