// ai Service 实现:每次调用从 biz_params 解析最新配置,admin 热更即生效。
package ai

import (
	"context"
	"strings"
)

// storeService 配置存储最小接口(PGStore 实现,测试可替换)。
type storeService interface {
	getParam(ctx context.Context, key string) (string, error)
	setParam(ctx context.Context, key, value string, updatedBy int64) error
}

type service struct {
	store storeService
}

// NewService 构造 Service;store 通常为 NewPGStore(pool)。
func NewService(store storeService) Service { return &service{store: store} }

// resolveConfig 读三键组装配置;apiKey/apiUrl 任一为空视为未配置。
func (s *service) resolveConfig(ctx context.Context) (Config, error) {
	var cfg Config
	var err error
	if cfg.APIURL, err = s.store.getParam(ctx, KeyAPIURL); err != nil {
		return cfg, err
	}
	if cfg.APIKey, err = s.store.getParam(ctx, KeyAPIKey); err != nil {
		return cfg, err
	}
	if cfg.Model, err = s.store.getParam(ctx, KeyModel); err != nil {
		return cfg, err
	}
	if cfg.APIURL == "" || cfg.APIKey == "" {
		return cfg, ErrNotConfigured
	}
	return cfg, nil
}

// ChatCompletion 对话补全;未配置返回 ErrNotConfigured,下游失败返回 ErrDownstream。
func (s *service) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if len(req.Messages) == 0 {
		return nil, ErrInvalidInput
	}
	cfg, err := s.resolveConfig(ctx)
	if err != nil {
		return nil, err
	}
	return chatCompletion(ctx, cfg, req)
}

// Embeddings 文本向量化;空输入直接返回空结果(不打下游)。
func (s *service) Embeddings(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	if len(req.Input) == 0 {
		return nil, ErrInvalidInput
	}
	cfg, err := s.resolveConfig(ctx)
	if err != nil {
		return nil, err
	}
	return embeddings(ctx, cfg, req)
}

// ConfigView 脱敏配置视图。
func (s *service) ConfigView(ctx context.Context) (ConfigView, error) {
	cfg, err := s.resolveConfigIgnoreMissing(ctx)
	if err != nil {
		return ConfigView{}, err
	}
	return ConfigView{
		APIURL:       cfg.APIURL,
		APIKeyMasked: maskKey(cfg.APIKey),
		Model:        cfg.Model,
		Configured:   cfg.APIURL != "" && cfg.APIKey != "",
	}, nil
}

// UpdateConfig 按 ConfigUpdate 逐键落库(nil 跳过,空串允许清空)。
func (s *service) UpdateConfig(ctx context.Context, upd ConfigUpdate, updatedBy int64) error {
	if upd.APIURL != nil {
		if !strings.HasPrefix(*upd.APIURL, "http") {
			return ErrInvalidInput
		}
		if err := s.store.setParam(ctx, KeyAPIURL, *upd.APIURL, updatedBy); err != nil {
			return err
		}
	}
	if upd.APIKey != nil {
		if err := s.store.setParam(ctx, KeyAPIKey, *upd.APIKey, updatedBy); err != nil {
			return err
		}
	}
	if upd.Model != nil {
		if err := s.store.setParam(ctx, KeyModel, *upd.Model, updatedBy); err != nil {
			return err
		}
	}
	return nil
}

// resolveConfigIgnoreMissing 与 resolveConfig 相同但不报未配置错误。
func (s *service) resolveConfigIgnoreMissing(ctx context.Context) (Config, error) {
	cfg, err := s.resolveConfig(ctx)
	if err == ErrNotConfigured {
		return cfg, nil
	}
	return cfg, err
}

// maskKey 密钥脱敏:保留前 4 后 4,长度不足 12 全掩码。
func maskKey(k string) string {
	if k == "" {
		return ""
	}
	if len(k) < 12 {
		return strings.Repeat("*", len(k))
	}
	return k[:4] + "****" + k[len(k)-4:]
}
