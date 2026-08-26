package stripe

// Webhook endpoint 管理(自愈/自检):列表/更新/创建。
// 契约命名红线:渠道原始 snake_case 响应不经 struct tag,一律 map 取字段。

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// WebhookEndpoint 已注册的 webhook endpoint;secret 仅在创建响应一次性返回。
type WebhookEndpoint struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// ListWebhookEndpoints 列出全部 endpoint(limit 100 足够单账号)。
func (c *Client) ListWebhookEndpoints(ctx context.Context) ([]WebhookEndpoint, error) {
	var raw map[string]any
	if err := c.getJSON(ctx, "/v1/webhook_endpoints?limit=100", &raw); err != nil {
		return nil, err
	}
	data, _ := raw["data"].([]any)
	out := make([]WebhookEndpoint, 0, len(data))
	for _, item := range data {
		m, _ := item.(map[string]any)
		out = append(out, WebhookEndpoint{ID: toStr(m["id"]), URL: toStr(m["url"])})
	}
	return out, nil
}

// UpdateWebhookEndpoint 更新 endpoint URL;签名密钥随 endpoint 保持(不换 whsec)。
func (c *Client) UpdateWebhookEndpoint(ctx context.Context, id, endpointURL string) error {
	form := url.Values{}
	form.Set("url", endpointURL)
	_, err := c.postForm(ctx, "/v1/webhook_endpoints/"+id, form, "")
	return err
}

// CreateWebhookEndpoint 创建 endpoint(url + enabled events),返回一次性 signing secret;
// 幂等键=endpointURL,重试不产生重复 endpoint。
func (c *Client) CreateWebhookEndpoint(ctx context.Context, endpointURL string, events []string) (WebhookEndpoint, string, error) {
	form := url.Values{}
	form.Set("url", endpointURL)
	for _, e := range events {
		form.Add("enabled_events[]", e)
	}
	body, err := c.postForm(ctx, "/v1/webhook_endpoints", form, endpointURL)
	if err != nil {
		return WebhookEndpoint{}, "", err
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return WebhookEndpoint{}, "", fmt.Errorf("stripe: decode endpoint: %w", err)
	}
	return WebhookEndpoint{ID: toStr(raw["id"]), URL: toStr(raw["url"])}, toStr(raw["secret"]), nil
}
