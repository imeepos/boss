package app

// W8 环节自动化测试:段内顺序推进 + 每环节发布状态变更事件。

import (
	"context"
	"testing"

	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/events"
)

type autoOrder struct {
	order.OrderService
	calls []string
	fail  map[string]bool
	track *order.Order
}

func (a *autoOrder) ApplyTag(context.Context, int64) error {
	a.calls = append(a.calls, "applyTag")
	return nil
}
func (a *autoOrder) CreateUserProfile(context.Context, int64) error {
	a.calls = append(a.calls, "createUserProfile")
	return nil
}
func (a *autoOrder) PreConfigOLT(context.Context, int64) error {
	a.calls = append(a.calls, "preConfigOLT")
	return nil
}
func (a *autoOrder) DispatchOrder(context.Context, int64) error {
	a.calls = append(a.calls, "dispatchOrder")
	return nil
}
func (a *autoOrder) ActivateUser(context.Context, int64) error {
	a.calls = append(a.calls, "activateUser")
	return nil
}
func (a *autoOrder) NotifyActivation(context.Context, int64) error {
	a.calls = append(a.calls, "notifyActivation")
	return nil
}
func (a *autoOrder) UpdateMap(context.Context, int64) error {
	a.calls = append(a.calls, "updateMap")
	return nil
}
func (a *autoOrder) Track(context.Context, int64) (*order.Order, []order.StageLog, error) {
	if a.track == nil {
		a.track = &order.Order{OrderNo: "ORD-T"}
	}
	return a.track, nil, nil
}

type capPub struct {
	events.Publisher
	got []events.Event
}

func (c *capPub) Publish(_ context.Context, _ string, e events.Event) error {
	c.got = append(c.got, e)
	return nil
}

func TestAutomation(t *testing.T) {
	t.Run("AutoPreScan 推进 5-8 且逐环节发事件", func(t *testing.T) {
		o, pub := &autoOrder{}, &capPub{}
		m := NewAutomation(o, pub)
		if err := m.AutoPreScan(context.Background(), 1); err != nil {
			t.Fatal(err)
		}
		if len(o.calls) != 4 {
			t.Fatalf("calls=%v", o.calls)
		}
		if len(pub.got) != 4 || pub.got[0].Type != "order.stage.changed" {
			t.Fatalf("events=%+v", pub.got)
		}
	})

	t.Run("AutoPostScan 推进 10-12", func(t *testing.T) {
		o, pub := &autoOrder{}, &capPub{}
		m := NewAutomation(o, pub)
		if err := m.AutoPostScan(context.Background(), 1); err != nil {
			t.Fatal(err)
		}
		if len(o.calls) != 3 || o.calls[2] != "updateMap" {
			t.Fatalf("calls=%v", o.calls)
		}
	})

	// ISSUE.md:worker 激活只推段10后,admin 重调 AutoPostScan 应跳过段10 从段11 续推(幂等)。
	t.Run("段10已完成 → AutoPostScan 跳过激活续推 11-12", func(t *testing.T) {
		o, pub := &autoOrder{}, &capPub{}
		o.track = &order.Order{OrderNo: "ORD-T", Stage: 10}
		m := NewAutomation(o, pub)
		if err := m.AutoPostScan(context.Background(), 1); err != nil {
			t.Fatal(err)
		}
		if len(o.calls) != 2 || o.calls[0] != "notifyActivation" || o.calls[1] != "updateMap" {
			t.Fatalf("calls=%v, want [notifyActivation updateMap]", o.calls)
		}
	})

	// 12 环节全部完成时重调为 no-op(重试幂等)。
	t.Run("段12已完成 → AutoPostScan 全跳过", func(t *testing.T) {
		o := &autoOrder{}
		o.track = &order.Order{OrderNo: "ORD-T", Stage: 12}
		m := NewAutomation(o, nil)
		if err := m.AutoPostScan(context.Background(), 1); err != nil {
			t.Fatal(err)
		}
		if len(o.calls) != 0 {
			t.Fatalf("calls=%v, want empty", o.calls)
		}
	})

	t.Run("nil publisher 降级不 panic", func(t *testing.T) {
		o := &autoOrder{}
		m := NewAutomation(o, nil)
		if err := m.AutoPreScan(context.Background(), 1); err != nil {
			t.Fatal(err)
		}
	})
}
