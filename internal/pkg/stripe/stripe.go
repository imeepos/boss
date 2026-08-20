// Package stripe Stripe 支付薄 HTTP 适配器(裁定 2026-08-20:不 vendor SDK,接口形态对齐
// billing.PaymentGateway 的 CreateIntent;凭据走环境变量引用,不落库)。
package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL Stripe 官方 API 地址;测试经 NewWithBaseURL 注入 httptest 服务。
const DefaultBaseURL = "https://api.stripe.com"

// Client Stripe REST 客户端(Bearer 密钥 + form-encoded 请求)。
type Client struct {
	APIKey   string
	BaseURL  string
	Currency string // 记账币种(如 php),用于意图金额语义,创建时由调用方传 cents
	HTTP     *http.Client
}

// New 构造客户端;apiKey 为空返回 nil(装配层据此判定通道未启用)。
func New(apiKey, currency string) *Client {
	if apiKey == "" {
		return nil
	}
	return &Client{APIKey: apiKey, BaseURL: DefaultBaseURL, Currency: currency, HTTP: &http.Client{Timeout: 10 * time.Second}}
}

// Channel 渠道标识(对账批次 channel 字段与 payments.method=card 并存)。
func (c *Client) Channel() string { return "stripe" }

// Intent 创建结果。
type Intent struct {
	ID           string
	ClientSecret string
	Status       string
	AmountCents  int64
	Currency     string
}

// CreateIntent 创建 PaymentIntent;payNo 作为幂等键(Idempotency-Key)与 metadata.pay_no,
// 回调侧凭 metadata 对账回填,pay_no 唯一约束保证落账幂等。
// 额外 metadata(kv)随意图持久化(如 bill_no/customer_id,回调落账寻址用)。
func (c *Client) CreateIntent(ctx context.Context, payNo string, amountCents int64, metadata map[string]string) (Intent, error) {
	form := url.Values{}
	form.Set("amount", strconv.FormatInt(amountCents, 10))
	form.Set("currency", c.Currency)
	form.Set("metadata[pay_no]", payNo)
	for k, v := range metadata {
		form.Set("metadata["+k+"]", v)
	}
	body, err := c.postForm(ctx, "/v1/payment_intents", form, payNo)
	if err != nil {
		return Intent{}, err
	}
	// 契约命名红线:不经 struct tag 解码渠道原始 snake_case 字段。
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return Intent{}, fmt.Errorf("stripe: decode intent: %w", err)
	}
	return Intent{
		ID:           toStr(raw["id"]),
		ClientSecret: toStr(raw["client_secret"]),
		Status:       toStr(raw["status"]),
		AmountCents:  toInt(raw["amount"]),
		Currency:     toStr(raw["currency"]),
	}, nil
}

// CheckoutSession 托管收银台会话(免客户端 SDK:前端/Android 直接跳 url)。
type CheckoutSession struct {
	ID  string
	URL string // 收银台跳转地址
}

// CreateCheckoutSession 创建 Checkout Session(mode=payment);metadata 透传至底层
// PaymentIntent,支付完成的 payment_intent.succeeded 回调沿用同一落账链路。
func (c *Client) CreateCheckoutSession(ctx context.Context, payNo string, amountCents int64,
	metadata map[string]string, successURL, cancelURL string) (CheckoutSession, error) {
	form := url.Values{}
	form.Set("mode", "payment")
	form.Set("success_url", successURL)
	form.Set("cancel_url", cancelURL)
	form.Set("line_items[0][quantity]", "1")
	form.Set("line_items[0][price_data][currency]", c.Currency)
	form.Set("line_items[0][price_data][unit_amount]", strconv.FormatInt(amountCents, 10))
	form.Set("line_items[0][price_data][product_data][name]", "BOSS Bill "+payNo)
	form.Set("metadata[pay_no]", payNo)
	for k, v := range metadata {
		form.Set("metadata["+k+"]", v)
	}
	body, err := c.postForm(ctx, "/v1/checkout/sessions", form, payNo)
	if err != nil {
		return CheckoutSession{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return CheckoutSession{}, fmt.Errorf("stripe: decode session: %w", err)
	}
	return CheckoutSession{ID: toStr(raw["id"]), URL: toStr(raw["url"])}, nil
}

func toStr(v any) string { s, _ := v.(string); return s }
func toInt(v any) int64  { f, _ := v.(float64); return int64(f) }

// postForm 发 form-encoded POST 并读响应体;非 2xx 返回带状态码与正文的错误。
func (c *Client) postForm(ctx context.Context, path string, form url.Values, idemKey string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+path,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stripe: post %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("stripe: read %s: %w", path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("stripe: %s status %d: %s", path, resp.StatusCode, truncate(body))
	}
	return body, nil
}

func truncate(b []byte) string {
	s := string(b)
	if len(s) > 300 {
		s = s[:300]
	}
	return s
}
