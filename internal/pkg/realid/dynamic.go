package realid

// Dynamic 按 resolve 回调懒加载通道配置(60s 缓存),配置源(biz_params)变更后热生效。
// 装配层提供 resolve(读 DB + 解密 + env 兜底);与 sms.Dynamic 同构,各通道自持不跨包抽公共层。

import (
	"context"
	"sync"
	"time"
)

// ChannelConfig 通道配置(resolve 返回的明文形态)。
type ChannelConfig struct {
	Enabled         bool
	AccessKeyID     string
	AccessKeySecret string
	Endpoint        string
}

type cachedVerifier struct {
	verifier Verifier
	at       time.Time
}

// Dynamic 配置懒加载核验通道:未启用或凭据缺失返回 ErrDisabled(提交保持 PENDING 人工核验)。
type Dynamic struct {
	resolve func(ctx context.Context) (ChannelConfig, error)
	mu      sync.Mutex
	cached  cachedVerifier
	ttl     time.Duration
}

// NewDynamic 构造动态通道;resolve 失败时本次核验直接报错(下次重试)。
func NewDynamic(resolve func(ctx context.Context) (ChannelConfig, error)) Verifier {
	return &Dynamic{resolve: resolve, ttl: 60 * time.Second}
}

// Verify 取当前配置通道核验;配置缓存 60s。
func (d *Dynamic) Verify(ctx context.Context, name, idNo string) (string, error) {
	v, err := d.current(ctx)
	if err != nil {
		return "", err
	}
	return v.Verify(ctx, name, idNo)
}

func (d *Dynamic) current(ctx context.Context) (Verifier, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cached.verifier != nil && time.Since(d.cached.at) < d.ttl {
		return d.cached.verifier, nil
	}
	cfg, err := d.resolve(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, ErrDisabled
	}
	v := NewAliyunCloudauth(AliyunCloudauthConfig{
		AccessKeyID:     cfg.AccessKeyID,
		AccessKeySecret: cfg.AccessKeySecret,
		Endpoint:        cfg.Endpoint,
	})
	if v == nil {
		return nil, ErrDisabled
	}
	d.cached = cachedVerifier{verifier: v, at: time.Now()}
	return v, nil
}
