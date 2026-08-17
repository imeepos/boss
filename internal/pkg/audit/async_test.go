package audit

import (
	"context"
	"sync"
	"testing"
)

// fakeWriter 桩 Writer:内存收集事件,供 AsyncWriter 测试。
type fakeWriter struct {
	mu     sync.Mutex
	events []Event
}

func (f *fakeWriter) Write(ctx context.Context, e Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
	return nil
}
func (f *fakeWriter) List(context.Context, Query) ([]Entry, error) { return nil, nil }
func (f *fakeWriter) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

// TestAsyncWriter 契约:Write 非阻塞入队,Close 排空后落库,幂等。
func TestAsyncWriter(t *testing.T) {
	f := &fakeWriter{}
	a := NewAsyncWriter(f, 16)

	for i := 0; i < 5; i++ {
		if err := a.Write(context.Background(), Event{TargetType: "order", Action: "状态变更"}); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	a.Close()
	a.Close() // 幂等

	if got := f.count(); got != 5 {
		t.Fatalf("flush count=%d, want 5", got)
	}
}
