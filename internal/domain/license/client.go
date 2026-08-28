package license

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient release-platform REST 客户端(激活链路专用:兑码 → 取离线令牌)。
// 端点与载荷对齐 release-platform api/openapi/openapi.yaml；
// 认证用 API token(bearer),由部署配置注入。
type APIClient struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
	// Tenant release-platform 租户 ID(JWT 模式从 claim 取,API token 模式下建 token 时的租户)。
	Tenant string
	// PublicKeyHex 内嵌公钥(离线令牌品相校验,与 Verifier 同源)。
	PublicKeyHex string
}

// NewAPIClient 构造客户端;HTTP 缺省 10s 超时。
func NewAPIClient(baseURL, token, tenant, publicKeyHex string) *APIClient {
	return &APIClient{
		BaseURL:      baseURL,
		Token:        token,
		Tenant:       tenant,
		PublicKeyHex: publicKeyHex,
		HTTP:         &http.Client{Timeout: 15 * time.Second},
	}
}

// 外部契约编解码红线(对齐 internal/pkg/stripe/source.go 先例):
// release-platform 载荷为 snake_case,外部字段名不经 struct tag,
// 请求体用 map 构造、响应体 map 取键。本文件因此不出现 snake_case json tag。

// mb 构造激活请求体(release-platform ActivationRequest)。
func activationReqBody(code, productID, deviceID, fingerprint string) map[string]string {
	return map[string]string{
		"activation_code":  code,
		"product_id":       productID,
		"device_id":        deviceID,
		"fingerprint_hash": fingerprint,
	}
}

// mapStr map 安全取值(string;缺失/类型不符返回空)。
func mapStr(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

// Exchange 完整激活:① POST /v1/activations 兑码 → ② POST /v1/licenses/{id}/offline-token
// 取离线令牌。返回 claims 与令牌原文(供本地落盘)。
func (c *APIClient) Exchange(ctx context.Context, activationCode, productID, deviceID, fingerprint string) (*Claims, []byte, error) {
	licID, err := c.activate(ctx, activationCode, productID, deviceID, fingerprint)
	if err != nil {
		return nil, nil, err
	}
	tok, err := c.offlineToken(ctx, licID, deviceID, fingerprint)
	if err != nil {
		return nil, nil, err
	}
	// 校验品相:令牌结构 + 签名(防止远端配置错公钥/被篡改)。
	v, err := NewVerifier(c.PublicKeyHex)
	if err != nil {
		return nil, nil, fmt.Errorf("license: %w", err)
	}
	// 只 marshal 令牌本体 {payload,key_id,signature,algorithm}:Verify 严格解析
	// (DisallowUnknownFields),外层响应的 license_id/expires_at 等会被拒——
	// 2026-08-27 实测 bug(整个 response 拿去验签,激活全链路 502)。
	data, err := json.Marshal(tok)
	if err != nil {
		return nil, nil, err
	}
	claims, err := v.Verify(data, VerifyOptions{Now: time.Now().UTC()})
	if err != nil {
		return nil, nil, fmt.Errorf("license: offline token %w", err)
	}
	return claims, data, nil
}

// activate POST /v1/activations,返回 license id。
func (c *APIClient) activate(ctx context.Context, code, productID, deviceID, fingerprint string) (string, error) {
	body := activationReqBody(code, productID, deviceID, fingerprint)
	var out map[string]any
	if err := c.do(ctx, http.MethodPost, "/v1/activations", body, &out); err != nil {
		return "", err
	}
	licID := mapStr(out, "license_id")
	if licID == "" {
		return "", fmt.Errorf("license: activation returned empty license_id")
	}
	return licID, nil
}

// offlineToken POST /v1/licenses/{id}/offline-token,返回令牌本体。
func (c *APIClient) offlineToken(ctx context.Context, licenseID, deviceID, fingerprint string) (Token, error) {
	body := map[string]string{"device_id": deviceID, "fingerprint_hash": fingerprint}
	path := fmt.Sprintf("/v1/licenses/%s/offline-token", licenseID)
	var out map[string]any
	if err := c.do(ctx, http.MethodPost, path, body, &out); err != nil {
		return Token{}, err
	}
	tokRaw, _ := out["token"].(map[string]any)
	tokBytes, err := json.Marshal(tokRaw)
	if err != nil {
		return Token{}, err
	}
	var tok Token
	if err := json.Unmarshal(tokBytes, &tok); err != nil {
		return Token{}, err
	}
	if tok.Payload == "" || tok.Signature == "" {
		return Token{}, fmt.Errorf("license: offline token incomplete")
	}
	return tok, nil
}

// do 统一请求:bearer 认证 + 状态码/错误码解析,失败留可 grep 日志载荷。
func (c *APIClient) do(ctx context.Context, method, path string, body, out any) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("license: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("license: read resp: %w", err)
	}
	if resp.StatusCode >= 400 {
		// 错误体 {code,message,trace_id} 经 map 取键,失败不静默。
		var ae map[string]any
		_ = json.Unmarshal(raw, &ae)
		return fmt.Errorf("license: %s %s -> %d %s (%s)", method, path, resp.StatusCode, mapStr(ae, "message"), mapStr(ae, "trace_id"))
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("license: decode %s: %w", path, err)
		}
	}
	return nil
}
