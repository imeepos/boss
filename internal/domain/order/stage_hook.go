// 订单环节推进的对外广播钩子(Q4 开放平台 M5:业务事件挂接 Webhook)。
// order 域不依赖 openplat(避免域间耦合),app 装配层把 WebhookDispatcher 注入进来。
package order

import "context"

// StageNotifier 环节推进广播:advance 成功后按 (orderNo, stage) 幂等发事件。
type StageNotifier interface {
	Emit(ctx context.Context, eventType, eventID string, payload any) (int, error)
}

// StageEventPayload order.stage.done 事件负载(开放面最小字段,与 open 端订单投影同口径)。
type StageEventPayload struct {
	OrderNo string `json:"orderNo"`
	Stage   int8   `json:"stage"`
	Status  string `json:"status"`
}

// StageEventType 事件类型常量(订阅中心登记用)。
const StageEventType = "order.stage.done"
