package app

// W8 环节自动化:环节 6/7/10/11 自动推进 + 每次迁移发布 Kafka 状态变更事件。
// 验收口径:人工只做收费(4)、扫码(9)与装维(8 派单确认);其余环节由本编排全自动。

import (
	"context"
	"fmt"
	"time"

	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/events"
)

// Automation 订单环节自动编排器。
type Automation struct {
	Order order.OrderService
	Pub   events.Publisher // nil 视为 Noop
}

// NewAutomation 构造;pub 为 nil 时事件丢弃(链路降级不影响推进)。
func NewAutomation(o order.OrderService, pub events.Publisher) *Automation {
	if pub == nil {
		pub = events.Noop{}
	}
	return &Automation{Order: o, Pub: pub}
}

// AutoPreScan 收费后→扫码前的自动段:环节 5 标签预绑定、6 建档、7 预配置、8 派单。
func (m *Automation) AutoPreScan(ctx context.Context, orderID int64) error {
	steps := []struct {
		event string
		run   func(context.Context, int64) error
	}{
		{"applyTag", m.Order.ApplyTag},
		{"createUserProfile", m.Order.CreateUserProfile},
		{"preConfigOLT", m.Order.PreConfigOLT},
		{"dispatchOrder", m.Order.DispatchOrder},
	}
	return m.run(ctx, orderID, steps)
}

// AutoPostScan 扫码(9)后的自动段:环节 10 激活、11 通知、12 上图。
func (m *Automation) AutoPostScan(ctx context.Context, orderID int64) error {
	steps := []struct {
		event string
		run   func(context.Context, int64) error
	}{
		{"activateUser", m.Order.ActivateUser},
		{"notifyActivation", m.Order.NotifyActivation},
		{"updateMap", m.Order.UpdateMap},
	}
	return m.run(ctx, orderID, steps)
}

// run 顺序推进并逐环节发事件;失败即停(调用方重试从失败环节续推,顺序守卫保证幂等)。
func (m *Automation) run(ctx context.Context, orderID int64, steps []struct {
	event string
	run   func(context.Context, int64) error
}) error {
	for i, st := range steps {
		if err := st.run(ctx, orderID); err != nil {
			return fmt.Errorf("automation: %s: %w", st.event, err)
		}
		m.emit(ctx, orderID, int8(i), st.event)
	}
	return nil
}

// emit 发布环节迁移事件(尽力而为,失败不影响推进)。
func (m *Automation) emit(ctx context.Context, orderID int64, idx int8, event string) {
	o, _, err := m.Order.Track(ctx, orderID)
	if err != nil || o == nil {
		return
	}
	_ = m.Pub.Publish(ctx, o.OrderNo, events.Event{
		Type: "order.stage.changed", OrderID: orderID, Stage: o.Stage, Status: o.Status,
		Timestamp: time.Now(),
	})
}
