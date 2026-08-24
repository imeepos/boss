package app

import (
	"context"
	"testing"
	"time"
)

// TestStartupDelaysDistinct 错峰约束:延迟互不相同、非负且有界,保证启动期首轮分散。
func TestStartupDelaysDistinct(t *testing.T) {
	seen := map[time.Duration]string{}
	for name, d := range startupDelays {
		if d < 0 || d > 2*time.Minute {
			t.Fatalf("%s delay=%v out of range", name, d)
		}
		if other, dup := seen[d]; dup {
			t.Fatalf("%s and %s share delay %v", other, name, d)
		}
		seen[d] = name
	}
	if len(startupDelays) < 2 {
		t.Fatal("expected multiple staggered loops")
	}
}

// TestStaggeredFirstRunRunsAfterOffset 首轮在 offset 之后才执行。
func TestStaggeredFirstRunRunsAfterOffset(t *testing.T) {
	ctx := context.Background()
	ran := make(chan time.Duration, 1)
	start := time.Now()
	if !staggeredFirstRun(ctx, 30*time.Millisecond, func(context.Context) {
		ran <- time.Since(start)
	}) {
		t.Fatal("expected first run to execute")
	}
	if d := <-ran; d < 30*time.Millisecond {
		t.Fatalf("first run after %v, want >= 30ms", d)
	}
}

// TestStaggeredFirstRunImmediate 零延迟保持原"启动即首轮"行为。
func TestStaggeredFirstRunImmediate(t *testing.T) {
	called := false
	if !staggeredFirstRun(context.Background(), 0, func(context.Context) { called = true }) {
		t.Fatal("zero offset should run immediately")
	}
	if !called {
		t.Fatal("fn not called")
	}
}

// TestStaggeredFirstRunCancelled 取消先到时不执行首轮。
func TestStaggeredFirstRunCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if staggeredFirstRun(ctx, time.Hour, func(context.Context) { t.Fatal("should not run") }) {
		t.Fatal("expected cancel to suppress first run")
	}
}
