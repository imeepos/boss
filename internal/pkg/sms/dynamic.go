package sms

// Dynamic 按 resolve 回调懒加载通道配置(60s 缓存),配置源(biz_params)变更后热生效。
// 装配层提供 resolve(读 DB + 解密 + env 兜底);本包不依赖具体存储。

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrDisabled 通道被管理员停用(sms.enabled=false)。
var ErrDisabled = errors.New("sms: channel disabled")

// ChannelConfig 通道配置(resolve 返回的明文形态)。
// ContentCodeCN/MY 为阿里云国际控制台报备模板编号(SendSms ContentCode)。
type ChannelConfig struct {
	Enabled         bool
	AccessKeyID     string
	AccessKeySecret string
	ContentCodeCN   string
	ContentCodeMY   string
}

type cachedSender struct {
	sender Sender
	at     time.Time
}

// Dynamic 配置懒加载通道:Enabled=false 报错;凭据齐备走阿里云国际;否则降级日志通道。
type Dynamic struct {
	resolve func(ctx context.Context) (ChannelConfig, error)
	mu      sync.Mutex
	cached  cachedSender
	ttl     time.Duration
}

// NewDynamic 构造动态通道;resolve 失败时本次发送直接报错(下次重试)。
func NewDynamic(resolve func(ctx context.Context) (ChannelConfig, error)) Sender {
	return &Dynamic{resolve: resolve, ttl: 60 * time.Second}
}

// Send 取当前配置通道发送;配置缓存 60s。
func (d *Dynamic) Send(ctx context.Context, phone, code, scene string) error {
	s, err := d.current(ctx)
	if err != nil {
		return err
	}
	return s.Send(ctx, phone, code, scene)
}

func (d *Dynamic) current(ctx context.Context) (Sender, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cached.sender != nil && time.Since(d.cached.at) < d.ttl {
		return d.cached.sender, nil
	}
	cfg, err := d.resolve(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, ErrDisabled
	}
	s := Sender(LogSender{})
	if cfg.AccessKeyID != "" && cfg.AccessKeySecret != "" {
		codes := map[string]string{}
		if cfg.ContentCodeCN != "" {
			codes["86"] = cfg.ContentCodeCN
		}
		if cfg.ContentCodeMY != "" {
			codes["60"] = cfg.ContentCodeMY
		}
		s = NewAliyunIntl(AliyunIntlConfig{
			AccessKeyID:     cfg.AccessKeyID,
			AccessKeySecret: cfg.AccessKeySecret,
			ContentCodes:    codes,
		})
	}
	d.cached = cachedSender{sender: s, at: time.Now()}
	return s, nil
}
