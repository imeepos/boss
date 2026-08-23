// 开放平台管理面路由(挂 admin 鉴权组,menu:openplat):
// 应用凭证 CRUD + Webhook 订阅管理。Secret 仅创建时返回一次。
package adminapi

import (
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
)

// registerOpenPlatRoutes 注册开放平台管理路由(menu:openplat)。
func registerOpenPlatRoutes(g *gin.RouterGroup, a *app.Application) {
	op := g.Group("", requirePerm(a.User, "menu:openplat"))
	op.GET("/openplat/apps", openPlatAppListHandler(a))
	op.POST("/openplat/apps", openPlatAppCreateHandler(a))
	op.PUT("/openplat/apps/:id/status", openPlatAppStatusHandler(a))
	op.GET("/openplat/apps/:id/subscriptions", openPlatSubListHandler(a))
	op.POST("/openplat/apps/:id/subscriptions", openPlatSubCreateHandler(a))
	op.DELETE("/openplat/subscriptions/:id", openPlatSubDeleteHandler(a))

	// M2:投递 outbox 管理面 + 集成方自助验收集(测试事件)。
	op.GET("/openplat/deliveries", openPlatDeliveryListHandler(a))
	op.POST("/openplat/deliveries/:id/requeue", openPlatDeliveryRequeueHandler(a))
	op.POST("/openplat/apps/:id/test-event", openPlatTestEventHandler(a))
}

// openPlatAppCreateReq 创建应用请求体(rpm/quota 缺省 60/10000)。
// name 由 gin binding:"required" 在反序列化阶段拒空字符串,
// 配合 validate() 里的 trim+长度校验,避免客户端发 "" 或全空白绕过。
type openPlatAppCreateReq struct {
	Name         string `json:"name" binding:"required"`
	RateLimitRPM int    `json:"rateLimitRpm"`
	DailyQuota   int    `json:"dailyQuota"`
	Sandbox      bool   `json:"sandbox"`
}

// openPlatAppStatusReq 启停应用请求体。
type openPlatAppStatusReq struct {
	Status int16 `json:"status" binding:"oneof=0 1"`
}

// openPlatSubCreateReq 新增 Webhook 订阅请求体。
type openPlatSubCreateReq struct {
	EventType   string `json:"eventType"`
	EndpointURL string `json:"endpointUrl"`
}

// validate 校验创建应用请求:trim 后非空 + 长度 ≤ 64;rpm/quota 非负。
func (r openPlatAppCreateReq) validate() error {
	name := strings.TrimSpace(r.Name)
	if name == "" {
		return &httpx.ValidationError{Field: "name", Message: "is required"}
	}
	if utf8.RuneCountInString(name) > 64 {
		return &httpx.ValidationError{Field: "name", Message: "max 64 characters"}
	}
	return httpx.CollectErrors(
		httpx.RequireNonNegativeFloat(float64(r.RateLimitRPM), "rateLimitRpm"),
		httpx.RequireNonNegativeFloat(float64(r.DailyQuota), "dailyQuota"),
	)
}

// validate 校验新增订阅请求。
func (r openPlatSubCreateReq) validate() error {
	return httpx.CollectErrors(
		httpx.RequireString(r.EventType, "eventType", 64),
		httpx.RequireString(r.EndpointURL, "endpointUrl", 512),
	)
}
