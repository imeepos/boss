// Package ai 将 openai-go SDK 能力封装为平台统一接口:
// apiKey/apiUrl 由 admin 集中配置(biz_params 热更),业务方无感知,密钥不外泄。
package ai

import (
	"context"
	"errors"
)

// 集中配置键(与 biz_params 对齐,admin 可经 PUT /api/v1/params/:key 或专用接口热更)。
const (
	KeyAPIURL = "ai.openai.apiUrl" // 如 https://api.openai.com/v1(需含 /v1)
	KeyAPIKey = "ai.openai.apiKey" // 平台级密钥,仅 admin 可见明文
	KeyModel  = "ai.openai.model"  // 默认模型,请求未指定 model 时兜底
)

// 领域错误。
var (
	ErrNotConfigured = errors.New("ai: openai apiKey/apiUrl not configured")
	ErrInvalidInput  = errors.New("ai: invalid input")
	ErrDownstream    = errors.New("ai: openai downstream error")
)

// ChatMessage 对话消息(role: system|user|assistant)。
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 对话补全请求;Model 空则用平台默认模型。
type ChatRequest struct {
	Model       string        `json:"model,omitempty"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int64         `json:"maxTokens,omitempty"`
}

// Usage token 用量。
type Usage struct {
	PromptTokens     int64 `json:"promptTokens"`
	CompletionTokens int64 `json:"completionTokens"`
	TotalTokens      int64 `json:"totalTokens"`
}

// ChatResponse 对话补全结果(单轮,取首个 choice)。
type ChatResponse struct {
	Model        string `json:"model"`
	Content      string `json:"content"`
	FinishReason string `json:"finishReason"`
	Usage        Usage  `json:"usage"`
}

// EmbeddingRequest 向量化请求;Model 空则用平台默认模型。
type EmbeddingRequest struct {
	Model string   `json:"model,omitempty"`
	Input []string `json:"input"`
}

// EmbeddingResponse 向量化结果,与 Input 顺序一一对应。
type EmbeddingResponse struct {
	Model   string      `json:"model"`
	Vectors [][]float64 `json:"vectors"`
	Usage   Usage       `json:"usage"`
}

// Config 平台级 OpenAI 连接配置(明文,仅域内使用)。
type Config struct {
	APIURL string
	APIKey string
	Model  string
}

// ConfigUpdate admin 配置变更(指针字段,nil=不修改)。
type ConfigUpdate struct {
	APIURL *string `json:"apiUrl"`
	APIKey *string `json:"apiKey"`
	Model  *string `json:"model"`
}

// ConfigView admin 可见的配置视图(apiKey 脱敏)。
type ConfigView struct {
	APIURL       string `json:"apiUrl"`
	APIKeyMasked string `json:"apiKeyMasked"` // 如 sk-****abcd,空=未配置
	Model        string `json:"model"`
	Configured   bool   `json:"configured"` // apiKey 与 apiUrl 均就绪
}

// Service OpenAI 能力统一接口:配置集中、调用方只关心业务参数。
type Service interface {
	// ChatCompletion 对话补全(封装 SDK Chat.Completions)。
	ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	// Embeddings 文本向量化(封装 SDK Embeddings)。
	Embeddings(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error)
	// ConfigView 读取脱敏配置(admin 展示)。
	ConfigView(ctx context.Context) (ConfigView, error)
	// UpdateConfig 变更集中配置(biz_params 热更,立即生效)。
	UpdateConfig(ctx context.Context, upd ConfigUpdate, updatedBy int64) error
}
