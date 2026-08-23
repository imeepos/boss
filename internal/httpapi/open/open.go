// 开放平台对外路由(M1,docs/plan/q4-open-platform-plan.md):
// 前缀 /api/open/v1,版本化;鉴权链为 HMAC 验签(OpenAuth)+ 每应用限流。
// 开放面只读或经内部服务校验,禁止直连核心事实表(季度红线)。
package openapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

// Register 注册开放平台路由:验签 → 限流 → 业务。
func Register(r *gin.Engine, a *app.Application) {
	limiter := middleware.NewOpenRateLimiter()
	v1 := r.Group("/api/open/v1")
	v1.Use(middleware.OpenAuth(a.OpenPlat), limiter.OpenRateLimit())
	registerOpenRoutes(v1, a)
}

func registerOpenRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/ping", openPingHandler())
	g.GET("/orders/:orderNo", openOrderHandler(a))
}

// openPingHandler 连通性自检:验签通过即返回应用行 ID。
func openPingHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0, "msg": "ok",
			"data": gin.H{"appRowId": middleware.OpenAppIDFrom(c), "version": "v1"},
		})
	}
}

// openOrderView 订单只读投影(开放面最小字段,不含内部成本口径)。
type openOrderView struct {
	OrderNo   string `json:"orderNo"`
	Status    string `json:"status"`
	Stage     int8   `json:"stage"`
	OfferID   int64  `json:"offerId"`
	CreatedAt string `json:"createdAt"`
}

// openOrderHandler 按订单号查询订单状态(只读,经 OrderService 读模型)。
func openOrderHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		o, err := a.Order.GetByNo(c.Request.Context(), c.Param("orderNo"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "order not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": openOrderView{
			OrderNo:   o.OrderNo,
			Status:    o.Status,
			Stage:     o.Stage,
			OfferID:   o.OfferID,
			CreatedAt: o.CreatedAt.Format(time.RFC3339),
		}})
	}
}
