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
	var raw struct {
		ID           string `json:"id"`
		ClientSecret string `json:"client_secret"`
		Status       string `json:"status"`
		Amount       int64  `json:"amount"`
		Currency     string `json:"currency"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return Intent{}, fmt.Errorf("stripe: decode intent: %w", err)
	}
	return Intent{ID: raw.ID, ClientSecret: raw.ClientSecret, Status: raw.Status,
		AmountCents: raw.Amount, Currency: raw.Currency}, nil
}

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
