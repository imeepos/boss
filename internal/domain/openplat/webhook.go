// Webhook 投递(M2):outbox + 指数退避重试 + 幂等。
// 事件发布方经 Emit 落 outbox(同订阅同 event_id 幂等去重);
// DeliverDue 由后台循环驱动,POST 到订阅端点并带 HMAC 负载签名,
// 2xx 视为成功;非 2xx/网络错误按退避重试,超过上限进死信。
package openplat

import (
	"context"
	"time"
)

// 投递状态枚举。
const (
	DeliveryPending = 0
	DeliveryDone    = 1
	DeliveryDead    = 2
)

// 投递参数。
const (
	MaxAttempts      = 6                // 超过即死信
	backoffBase      = 30 * time.Second // 首次重试延迟
	backoffMax       = 1 * time.Hour    // 退避上限
	DeliveryBatchMax = 20               // 每轮最多处理条数
)

// Delivery outbox 行(管理面视图)。
type Delivery struct {
	ID             int64  `json:"id"`
	SubscriptionID int64  `json:"subscriptionId"`
	EventID        string `json:"eventId"`
	EventType      string `json:"eventType"`
	Payload        string `json:"payload"` // JSON 文本
	Status         int16  `json:"status"`
	Attempts       int    `json:"attempts"`
	NextAttemptAt  string `json:"nextAttemptAt"`
	HTTPStatus     int    `json:"httpStatus"`
	LastError      string `json:"lastError"`
	DeliveredAt    string `json:"deliveredAt"`
	CreatedAt      string `json:"createdAt"`
}

// DueDelivery 待投递行(含端点与应用 Secret,投递器内部用)。
type DueDelivery struct {
	Delivery
	EndpointURL string
	Secret      string
}

// Emitter 事件发布接口(业务域注入:状态机关键节点调用)。
type Emitter interface {
	// Emit 广播事件到所有匹配的启用订阅;返回新建投递条数(幂等重放返回 0)。
	Emit(ctx context.Context, eventType, eventID string, payload any) (int, error)
}

// Deliverer 投递循环接口(app 层后台 loop 消费)。
type Deliverer interface {
	// DeliverDue 处理一批到期投递,返回处理条数。
	DeliverDue(ctx context.Context) (int, error)
}

// Poster HTTP 投递抽象(测试替换);nil 用默认 10s 超时 client。
type Poster interface {
	Post(url string, headers map[string]string, body []byte) (int, error)
}

// NewWebhookDispatcher 构造投递器。
func NewWebhookDispatcher(store WebhookStore, poster Poster) *WebhookDispatcher {
	return &WebhookDispatcher{store: store, poster: poster}
}

// WebhookStore outbox 存取接口(pg 实现)。
type WebhookStore interface {
	// InsertDeliveries 为匹配的启用订阅批量建投递行(ON CONFLICT DO NOTHING)。
	InsertDeliveries(ctx context.Context, eventType, eventID string, payload []byte) (int64, error)

	// InsertAppDeliveries 为指定应用的全部启用订阅批量建投递行(测试事件自检用:
	// 不按事件类型过滤;投递行事件类型记测试事件名,投递时 X-BOSS-Event 头即测试事件)。
	InsertAppDeliveries(ctx context.Context, appID int64, eventType, eventID string, payload []byte) (int64, error)

	// ListDue 原子领取一批到期投递(含端点/Secret):并发调用批次互不相交,
	// 领取时把 next_attempt_at 推进租约窗口,投递器崩溃后行自动重回待投。
	ListDue(ctx context.Context, now time.Time, limit int) ([]DueDelivery, error)

	// MarkResult 落投递结果:成功记 delivered;失败按退避排下次或进死信。
	MarkResult(ctx context.Context, id int64, success bool, httpStatus int, errMsg string) error

	// ListDeliveries 管理面查询(按订阅过滤,0=全部)。
	ListDeliveries(ctx context.Context, subscriptionID int64) ([]Delivery, error)

	// Requeue 手动重试死信/失败行(管理面)。
	Requeue(ctx context.Context, id int64) error
}

// WebhookDispatcher Emitter + Deliverer 实现。
type WebhookDispatcher struct {
	store  WebhookStore
	poster Poster
}

// Backoff 第 attempts 次失败后的下次尝试延迟(指数退避,封顶 1h)。
func Backoff(attempts int) time.Duration {
	d := backoffBase << (attempts - 1)
	if d > backoffMax || d <= 0 {
		return backoffMax
	}
	return d
}
