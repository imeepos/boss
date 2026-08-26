package stripe

// Dynamic 按 resolve 回调懒加载通道配置(60s 缓存),配置源(biz_params)变更后热生效。
// 装配层提供 resolve(读 DB + 解密 + env 兜底);与 sms/realid Dynamic 同构,各通道自持不跨包抽公共层。

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrDisabled 通道被管理员停用(stripe.enabled=false)。
var ErrDisabled = errors.New("stripe: channel disabled")

type cachedConfig struct {
	cfg    Config
	client *Client // 可能为 nil:配置有效但缺 APIKey
	at     time.Time
}

// Dynamic 配置懒加载卡收单通道:未启用视为未配置(调用方 400/503 降级)。
type Dynamic struct {
	resolve func(ctx context.Context) (Config, error)
	mu      sync.Mutex
	cached  cachedConfig
	ttl     time.Duration
}

// NewDynamic 构造动态通道;resolve 失败时本次调用报错(下次重试)。
func NewDynamic(resolve func(ctx context.Context) (Config, error)) *Dynamic {
	return &Dynamic{resolve: resolve, ttl: 60 * time.Second}
}

// Configured 通道是否可用(启用且 APIKey 已配)。
func (d *Dynamic) Configured(ctx context.Context) bool {
	cfg, _, err := d.current(ctx)
	return err == nil && cfg.APIKey != ""
}

// Client 返回当前凭据对应的客户端;未启用/缺凭据返回 ErrDisabled。
func (d *Dynamic) Client(ctx context.Context) (*Client, error) {
	_, c, err := d.current(ctx)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrDisabled
	}
	return c, nil
}

// WebhookSecret 返回当前回调验签密钥;未配置返回空串(与客户端构造解耦)。
func (d *Dynamic) WebhookSecret(ctx context.Context) string {
	cfg, _, err := d.current(ctx)
	if err != nil {
		return ""
	}
	return cfg.WebhookSecret
}

// Config 返回当前配置明文(自检/测试用)。
func (d *Dynamic) Config(ctx context.Context) (Config, error) {
	cfg, _, err := d.current(ctx)
	return cfg, err
}

func (d *Dynamic) current(ctx context.Context) (Config, *Client, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if time.Since(d.cached.at) < d.ttl {
		return d.cached.cfg, d.cached.client, nil
	}
	cfg, err := d.resolve(ctx)
	if err != nil {
		return Config{}, nil, err
	}
	if !cfg.Enabled {
		return Config{}, nil, ErrDisabled
	}
	d.cached = cachedConfig{cfg: cfg, at: time.Now()}
	if cfg.APIKey == "" {
		return cfg, nil, nil // 配置有效但无发起凭据
	}
	c := New(cfg.APIKey, cfg.Currency)
	if c == nil {
		return cfg, nil, nil
	}
	if cfg.APIBaseURL != "" {
		c.BaseURL = cfg.APIBaseURL
	}
	d.cached.client = c
	return cfg, c, nil
}
