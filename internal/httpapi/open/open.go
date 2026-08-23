// 开放平台对外路由(M1,docs/plan/q4-open-platform-plan.md):
// 前缀 /api/open/v1,版本化;鉴权链为 HMAC 验签(OpenAuth)+ 每应用限流。
// 开放面只读或经内部服务校验,禁止直连核心事实表(季度红线)。
package openapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/openplat"
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
	// 沙箱样例清单 + Webhook 验答回放样例(自助验收工具的数据源)。
	g.GET("/sandbox/samples", openSandboxSamplesHandler())
}

// openPingHandler 连通性自检:验签通过即返回应用行 ID。
func openPingHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0, "msg": "ok",
			"data": gin.H{
				"appRowId": middleware.OpenAppIDFrom(c),
				"sandbox":  middleware.OpenSandboxFrom(c),
				"version":  "v1",
			},
		})
	}
}

// openSandboxSamplesHandler 沙箱样例清单:样例订单号 + Webhook 验答回放样例。
func openSandboxSamplesHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0, "msg": "ok",
			"data": gin.H{
				"orders":  openplat.SandboxOrderNos(),
				"webhook": openplat.SandboxWebhookSample(),
			},
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
// 沙箱应用(M4)只返回 openplat 样例订单(SBX-*),与生产订单数据隔离;
// 沙箱应用查生产订单号一律 404(隔离即双向:见不到彼此)。
func openOrderHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderNo := c.Param("orderNo")
		if middleware.OpenSandboxFrom(c) {
			if raw, ok := openplat.SandboxOrderByNo(orderNo); ok {
				c.Data(http.StatusOK, "application/json", mustOpenJSON(raw))
				return
			}
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "order not found (sandbox)"})
			return
		}
		if openplat.IsSandboxOrderNo(orderNo) {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "order not found"})
			return
		}
		o, err := a.Order.GetByNo(c.Request.Context(), orderNo)
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

// mustOpenJSON 把样例 data 包进统一 envelope;失败即 panic(样例来自编译期字面量,不可能失败)。
func mustOpenJSON(raw json.RawMessage) []byte {
	env := gin.H{"code": 0, "msg": "ok", "data": raw}
	b, err := json.Marshal(env)
	if err != nil {
		panic(err)
	}
	return b
}
