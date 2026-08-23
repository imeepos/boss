// Package openplat 开放平台域(Q4 开放平台与互操作 M1):
// 外部集成方应用凭证、HMAC 请求签名验签、限流配额计数与 Webhook 事件订阅管理。
//
// 与 internal/domain/apikey 的边界:
//   - apikey 面向内部三端主体(account/worker/customer),bearer token,只存哈希;
//   - openplat 面向外部集成方应用,AppId+Secret+HMAC 签名,Secret 需原文落库验签
//     (不可逆决策,见 docs/notes/adopted/2026-08-22-open-platform-secret.md)。
package openplat

import (
	"context"
	"errors"
	"time"
)

// 错误。
var (
	ErrNotFound       = errors.New("openplat: app not found")
	ErrBadSignature   = errors.New("openplat: bad signature")
	ErrStaleTimestamp = errors.New("openplat: stale timestamp")
)

// TimestampWindow 防重放时间窗(秒)。
const TimestampWindow = 5 * 60

// App 应用凭证元数据(不含 Secret)。
type App struct {
	ID           int64  `json:"id"`
	AppID        string `json:"appId"`
	Name         string `json:"name"`
	Status       int16  `json:"status"` // 1启用 0停用
	RateLimitRPM int    `json:"rateLimitRpm"`
	DailyQuota   int    `json:"dailyQuota"`
	Sandbox      bool   `json:"sandbox"`
	LastUsedAt   string `json:"lastUsedAt"` // ISO8601,空=从未使用
	CreatedAt    string `json:"createdAt"`
}

// AppCreateResult 创建成功返回(Secret 仅此时返回一次)。
type AppCreateResult struct {
	App
	Secret string `json:"secret"`
}

// Subscription Webhook 事件订阅(M2 投递器消费;M1 先落管理面)。
type Subscription struct {
	ID          int64  `json:"id"`
	AppID       int64  `json:"appId"` // open_apps.id
	EventType   string `json:"eventType"`
	EndpointURL string `json:"endpointUrl"`
	Status      int16  `json:"status"`
	CreatedAt   string `json:"createdAt"`
}

// Service 开放平台管理接口。
type Service interface {
	// CreateApp 创建集成方应用,返回 AppID 与 Secret(Secret 仅一次)。
	CreateApp(ctx context.Context, name string, rateLimitRPM, dailyQuota int, sandbox bool, createdBy int64) (*AppCreateResult, error)

	// ListApps 列出全部应用元数据。
	ListApps(ctx context.Context) ([]App, error)

	// SetAppStatus 启用/停用应用。
	SetAppStatus(ctx context.Context, id int64, status int16) error

	// LookupActive 按公开 AppID 取启用中应用的验签上下文。
	LookupActive(ctx context.Context, appID string) (*AuthContext, error)

	// TouchUsage 记录一次成功调用:last_used_at + 当日计数自增,返回当日累计。
	TouchUsage(ctx context.Context, id int64, day time.Time) (int64, error)

	// CreateSubscription / ListSubscriptions / DeleteSubscription 管理 Webhook 订阅。
	CreateSubscription(ctx context.Context, appID int64, eventType, endpointURL string) (*Subscription, error)
	ListSubscriptions(ctx context.Context, appID int64) ([]Subscription, error)
	DeleteSubscription(ctx context.Context, id int64) error

	// ListDeliveries / Requeue 管理 Webhook 投递 outbox(M2,迁移 000125)。
	ListDeliveries(ctx context.Context, subscriptionID int64) ([]Delivery, error)
	Requeue(ctx context.Context, id int64) error
}

// AuthContext 验签所需的最小应用上下文。
type AuthContext struct {
	AppRowID     int64
	Secret       string
	RateLimitRPM int
	DailyQuota   int
	Sandbox      bool // 沙箱应用:开放面只见样例数据(M4)
}
