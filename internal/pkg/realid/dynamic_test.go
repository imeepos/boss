package realid

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// stubResolve 计数回调,模拟 biz_params 配置源。
func stubResolve(cfg ChannelConfig, err error, calls *int) func(context.Context) (ChannelConfig, error) {
	return func(context.Context) (ChannelConfig, error) {
		*calls++
		return cfg, err
	}
}

// TestDynamic_Disabled 未启用 → ErrDisabled(提交保持 PENDING 人工核验)。
func TestDynamic_Disabled(t *testing.T) {
	calls := 0
	d := NewDynamic(stubResolve(ChannelConfig{}, nil, &calls))
	if _, err := d.Verify(context.Background(), "张三", "x"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("want ErrDisabled, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("resolve calls=%d", calls)
	}
}

// TestDynamic_ResolveError 配置源读取失败 → 本次核验报错,不缓存(下次重试)。
func TestDynamic_ResolveError(t *testing.T) {
	boom := errors.New("db down")
	calls := 0
	d := NewDynamic(stubResolve(ChannelConfig{}, boom, &calls))
	if _, err := d.Verify(context.Background(), "张三", "x"); !errors.Is(err, boom) {
		t.Fatalf("want resolve error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("resolve calls=%d", calls)
	}
}

// TestDynamic_CacheAndTTL 命中缓存不重复 resolve;TTL 过期后重新 resolve。
func TestDynamic_CacheAndTTL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"Code":"200","ResultObject":{"BizCode":"1"}}`))
	}))
	defer srv.Close()

	calls := 0
	d := NewDynamic(stubResolve(ChannelConfig{
		Enabled: true, AccessKeyID: "ak", AccessKeySecret: "sk", Endpoint: srv.URL,
	}, nil, &calls)).(*Dynamic)

	for i := 1; i <= 2; i++ {
		got, err := d.Verify(context.Background(), "张三", "x")
		if err != nil || got != Pass {
			t.Fatalf("call %d: got=%s err=%v", i, got, err)
		}
	}
	if calls != 1 {
		t.Fatalf("cache miss: resolve calls=%d", calls)
	}

	// TTL 过期 → 重新 resolve 并重建通道。
	d.mu.Lock()
	d.cached.at = time.Now().Add(-2 * d.ttl)
	d.mu.Unlock()
	if _, err := d.Verify(context.Background(), "张三", "x"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("after TTL: resolve calls=%d", calls)
	}
}

// TestDynamic_MissingCreds 已启用但凭据缺失 → 通道构造为 nil,落 ErrDisabled。
func TestDynamic_MissingCreds(t *testing.T) {
	calls := 0
	d := NewDynamic(stubResolve(ChannelConfig{Enabled: true}, nil, &calls))
	if _, err := d.Verify(context.Background(), "张三", "x"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("want ErrDisabled, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("resolve calls=%d", calls)
	}
}

// TestDynamic_RebuildAfterDisabled 通道关闭后 TTL 过期 → 重新 resolve 返回未启用。
func TestDynamic_RebuildAfterDisabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"Code":"200","ResultObject":{"BizCode":"1"}}`))
	}))
	defer srv.Close()

	cfg := ChannelConfig{Enabled: true, AccessKeyID: "ak", AccessKeySecret: "sk", Endpoint: srv.URL}
	calls := 0
	on := true
	d := NewDynamic(func(context.Context) (ChannelConfig, error) {
		calls++
		if !on {
			cfg = ChannelConfig{}
		}
		return cfg, nil
	}).(*Dynamic)

	if _, err := d.Verify(context.Background(), "张三", "x"); err != nil {
		t.Fatal(err)
	}
	d.mu.Lock()
	d.cached.at = time.Now().Add(-2 * d.ttl)
	d.mu.Unlock()
	on = false
	if _, err := d.Verify(context.Background(), "张三", "x"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("want ErrDisabled after config flip, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("resolve calls=%d", calls)
	}
}
