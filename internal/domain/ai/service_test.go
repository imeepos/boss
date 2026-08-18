// ai Service 集成测试:httptest 模拟 OpenAI 兼容服务,验证配置解析/热更/脱敏与调用链路。
package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// memStore 内存配置存储,模拟 biz_params。
type memStore struct{ m map[string]string }

func (s *memStore) getParam(_ context.Context, key string) (string, error) {
	return s.m[key], nil
}
func (s *memStore) setParam(_ context.Context, key, value string, _ int64) error {
	s.m[key] = value
	return nil
}

// newMockOpenAI 起一个 OpenAI 兼容 mock:校验 Authorization,回固定 completion/embedding。
func newMockOpenAI(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer sk-test-123456" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "bad key"}})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/chat/completions":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "cmpl-1", "model": "gpt-test",
				"choices": []map[string]any{{
					"index": 0, "finish_reason": "stop",
					"message": map[string]any{"role": "assistant", "content": "hello from mock"},
				}},
				"usage": map[string]any{"prompt_tokens": 5, "completion_tokens": 3, "total_tokens": 8},
			})
		case "/v1/embeddings":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"model": "gpt-test",
				"data": []map[string]any{
					{"index": 0, "embedding": []float64{0.1, 0.2}},
					{"index": 1, "embedding": []float64{0.3, 0.4}},
				},
				"usage": map[string]any{"prompt_tokens": 4, "total_tokens": 4},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestService_NotConfigured(t *testing.T) {
	svc := NewService(&memStore{m: map[string]string{}})
	_, err := svc.ChatCompletion(context.Background(), ChatRequest{Messages: []ChatMessage{{Role: "user", Content: "hi"}}})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("want ErrNotConfigured, got %v", err)
	}
}

func TestService_ChatCompletion_UsesCentralConfig(t *testing.T) {
	mock := newMockOpenAI(t)
	store := &memStore{m: map[string]string{}}
	svc := NewService(store)

	// admin 集中配置 apiUrl/apiKey(经 UpdateConfig 热更)。
	url := mock.URL + "/v1"
	key := "sk-test-123456"
	if err := svc.UpdateConfig(context.Background(), ConfigUpdate{APIURL: &url, APIKey: &key, Model: strPtr("gpt-test")}, 1); err != nil {
		t.Fatalf("UpdateConfig: %v", err)
	}

	res, err := svc.ChatCompletion(context.Background(), ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if res.Content != "hello from mock" || res.Model != "gpt-test" || res.Usage.TotalTokens != 8 {
		t.Fatalf("unexpected res: %+v", res)
	}
}

func TestService_ConfigView_MaskedAndHotUpdate(t *testing.T) {
	mock := newMockOpenAI(t)
	store := &memStore{m: map[string]string{}}
	svc := NewService(store)
	url, key := mock.URL+"/v1", "sk-test-123456"
	_ = svc.UpdateConfig(context.Background(), ConfigUpdate{APIURL: &url, APIKey: &key}, 1)

	view, err := svc.ConfigView(context.Background())
	if err != nil {
		t.Fatalf("ConfigView: %v", err)
	}
	if !view.Configured || view.APIKeyMasked != "sk-t****3456" || strings.Contains(view.APIKeyMasked, "1234") {
		t.Fatalf("masked view wrong: %+v", view)
	}

	// 换成错误 key 后调用应报下游错误(401 透传为 ErrDownstream)。
	bad := "sk-wrong-key-0000"
	_ = svc.UpdateConfig(context.Background(), ConfigUpdate{APIKey: &bad}, 1)
	_, err = svc.ChatCompletion(context.Background(), ChatRequest{Model: "gpt-test", Messages: []ChatMessage{{Role: "user", Content: "hi"}}})
	if !errors.Is(err, ErrDownstream) {
		t.Fatalf("want ErrDownstream, got %v", err)
	}
}

func TestService_Embeddings(t *testing.T) {
	mock := newMockOpenAI(t)
	url, key := mock.URL+"/v1", "sk-test-123456"
	svc := NewService(&memStore{m: map[string]string{
		KeyAPIURL: url, KeyAPIKey: key, KeyModel: "gpt-test",
	}})
	res, err := svc.Embeddings(context.Background(), EmbeddingRequest{Input: []string{"a", "b"}})
	if err != nil {
		t.Fatalf("Embeddings: %v", err)
	}
	if len(res.Vectors) != 2 || res.Vectors[1][0] != 0.3 {
		t.Fatalf("unexpected vectors: %+v", res.Vectors)
	}
}

func TestService_NoModelAnywhere(t *testing.T) {
	mock := newMockOpenAI(t)
	url, key := mock.URL+"/v1", "sk-test-123456"
	svc := NewService(&memStore{m: map[string]string{KeyAPIURL: url, KeyAPIKey: key}}) // 无默认模型
	_, err := svc.ChatCompletion(context.Background(), ChatRequest{Messages: []ChatMessage{{Role: "user", Content: "hi"}}})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
	// 请求显式带 model 即可成功。
	res, err := svc.ChatCompletion(context.Background(), ChatRequest{Model: "gpt-test", Messages: []ChatMessage{{Role: "user", Content: "hi"}}})
	if err != nil || res.Content != "hello from mock" {
		t.Fatalf("explicit model: res=%+v err=%v", res, err)
	}
}

func strPtr(s string) *string { return &s }
