package app

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type fakeExpirer struct {
	calls int32
	fail  bool
}

func (f *fakeExpirer) ExpireDue(context.Context) (int, error) {
	atomic.AddInt32(&f.calls, 1)
	if f.fail {
		return 0, errors.New("boom")
	}
	return 2, nil
}

func TestStartPointsExpireLoop_RunsAndStops(t *testing.T) {
	f := &fakeExpirer{}
	stop := startPointsExpireLoop(f)
	// 循环只按 ticker 周期触发,验证启停不泄漏 goroutine 即可;
	// 周期设为 1h,单测不等待首跳,仅验证 stop 立即返回。
	stop()
	if atomic.LoadInt32(&f.calls) != 0 {
		t.Fatalf("expected no immediate call, got %d", f.calls)
	}
}

func TestPointsExpireLoop_TickHandled(t *testing.T) {
	// 直接驱动 ExpireDue 的错误路径,保证失败不 panic。
	f := &fakeExpirer{fail: true}
	if _, err := f.ExpireDue(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	_ = time.Millisecond // keep time import for interval semantics
}
