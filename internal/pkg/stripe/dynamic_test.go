package stripe

import (
	"context"
	"errors"
	"testing"
	"time"
)

// stubResolve 计数回调,模拟 biz_params 配置源。
func stubStripeResolve(cfg Config, err error, calls *int) func(context.Context) (Config, error) {
	return func(context.Context) (Config, error) {
		*calls++
		return cfg, err
	}
}

// TestDynamic_Disabled 未启用 → 未配置(Configured=false,Client=ErrDisabled)。
func TestDynamic_Disabled(t *testing.T) {
	calls := 0
	d := NewDynamic(stubStripeResolve(Config{}, nil, &calls))
	if d.Configured(context.Background()) {
		t.Fatal("disabled channel must not be configured")
	}
	if _, err := d.Client(context.Background()); !errors.Is(err, ErrDisabled) {
		t.Fatalf("want ErrDisabled, got %v", err)
	}
	if calls != 2 { // Configured+Client 各触发一次,ErrDisabled 不缓存
		t.Fatalf("resolve calls=%d", calls)
	}
}

// TestDynamic_MissingAPIKey 已启用但缺 APIKey → 同未配置。
func TestDynamic_MissingAPIKey(t *testing.T) {
	calls := 0
	d := NewDynamic(stubStripeResolve(Config{Enabled: true}, nil, &calls))
	if d.Configured(context.Background()) {
		t.Fatal("missing api key must not be configured")
	}
	if calls != 1 {
		t.Fatalf("resolve calls=%d", calls)
	}
}

// TestDynamic_ResolveError 配置源读取失败 → 本次调用报错,不缓存(下次重试)。
func TestDynamic_ResolveError(t *testing.T) {
	boom := errors.New("db down")
	calls := 0
	d := NewDynamic(stubStripeResolve(Config{}, boom, &calls))
	if d.Configured(context.Background()) {
		t.Fatal("resolve error must not configure")
	}
	if _, err := d.Client(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("want resolve error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("resolve calls=%d", calls)
	}
}

// TestDynamic_CacheAndTTL 命中缓存不重复 resolve;TTL 过期后重新 resolve。
func TestDynamic_CacheAndTTL(t *testing.T) {
	calls := 0
	d := NewDynamic(stubStripeResolve(Config{
		Enabled: true, APIKey: "sk_test_x", WebhookSecret: "whsec_x", Currency: "php",
	}, nil, &calls))

	ctx := context.Background()
	for i := 1; i <= 2; i++ {
		if !d.Configured(ctx) || d.WebhookSecret(ctx) != "whsec_x" {
			t.Fatalf("call %d: not configured/secret mismatch", i)
		}
	}
	if calls != 1 {
		t.Fatalf("cache miss: resolve calls=%d", calls)
	}

	d.mu.Lock()
	d.cached.at = time.Now().Add(-2 * d.ttl)
	d.mu.Unlock()
	if !d.Configured(ctx) {
		t.Fatal("after TTL should reconfigure")
	}
	if calls != 2 {
		t.Fatalf("after TTL: resolve calls=%d", calls)
	}
}

// TestDynamic_ConfigFlip 配置翻转(启用→停用)TTL 过期后按新配置生效。
func TestDynamic_ConfigFlip(t *testing.T) {
	calls := 0
	on := true
	cfg := Config{Enabled: true, APIKey: "sk_test_x", Currency: "php"}
	d := NewDynamic(func(context.Context) (Config, error) {
		calls++
		if !on {
			cfg = Config{}
		}
		return cfg, nil
	})

	ctx := context.Background()
	if !d.Configured(ctx) {
		t.Fatal("initial should be configured")
	}
	d.mu.Lock()
	d.cached.at = time.Now().Add(-2 * d.ttl)
	d.mu.Unlock()
	on = false
	if d.Configured(ctx) {
		t.Fatal("flip to disabled should deconfigure")
	}
	if calls != 2 {
		t.Fatalf("resolve calls=%d", calls)
	}
}
