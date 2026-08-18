// openai-go SDK 薄封装:按 Config 构造 client,域类型 <-> SDK 参数互转。
package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// newSDKClient 按配置构造 SDK client(每请求重建,配置热更即生效,client 构造无 IO 开销)。
func newSDKClient(cfg Config) *openai.Client {
	cl := openai.NewClient(
		option.WithBaseURL(strings.TrimRight(cfg.APIURL, "/")),
		option.WithAPIKey(cfg.APIKey),
	)
	return &cl
}

// chatCompletion 调用 SDK Chat.Completions,返回首个 choice。
func chatCompletion(ctx context.Context, cfg Config, req ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = cfg.Model
	}
	if model == "" {
		return nil, fmt.Errorf("%w: model 未指定且平台未配置默认模型", ErrInvalidInput)
	}
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(model),
		Messages: toSDKMessages(req.Messages),
	}
	if req.Temperature > 0 {
		params.Temperature = openai.Float(req.Temperature)
	}
	if req.MaxTokens > 0 {
		params.MaxTokens = openai.Int(req.MaxTokens)
	}
	resp, err := newSDKClient(cfg).Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDownstream, err.Error())
	}
	out := &ChatResponse{Model: resp.Model}
	if len(resp.Choices) > 0 {
		out.Content = resp.Choices[0].Message.Content
		out.FinishReason = resp.Choices[0].FinishReason
	}
	out.Usage = Usage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}
	return out, nil
}

// toSDKMessages 域消息转 SDK 联合类型;未知 role 按 user 处理。
func toSDKMessages(msgs []ChatMessage) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "system":
			out = append(out, openai.SystemMessage(m.Content))
		case "assistant":
			out = append(out, openai.AssistantMessage(m.Content))
		default:
			out = append(out, openai.UserMessage(m.Content))
		}
	}
	return out
}

// embeddings 调用 SDK Embeddings,向量与输入顺序一一对应。
func embeddings(ctx context.Context, cfg Config, req EmbeddingRequest) (*EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		model = cfg.Model
	}
	if model == "" {
		return nil, fmt.Errorf("%w: model 未指定且平台未配置默认模型", ErrInvalidInput)
	}
	resp, err := newSDKClient(cfg).Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(model),
		Input: openai.EmbeddingNewParamsInputUnion{OfArrayOfStrings: req.Input},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDownstream, err.Error())
	}
	out := &EmbeddingResponse{Model: resp.Model, Vectors: make([][]float64, len(resp.Data))}
	for _, d := range resp.Data {
		if d.Index >= 0 && int(d.Index) < len(out.Vectors) {
			vec := make([]float64, len(d.Embedding))
			for i, f := range d.Embedding {
				vec[i] = float64(f)
			}
			out.Vectors[d.Index] = vec
		}
	}
	out.Usage = Usage{
		PromptTokens: resp.Usage.PromptTokens,
		TotalTokens:  resp.Usage.TotalTokens,
	}
	return out, nil
}
