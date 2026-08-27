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

// activationRequest 兑码请求(release-platform ActivationRequest)。
type activationRequest struct {
	ActivationCode  string `json:"activation_code"`
	ProductID       string `json:"product_id"`
	DeviceID        string `json:"device_id"`
	FingerprintHash string `json:"fingerprint_hash"`
}

// activationResponse 兑码响应(release-platform ActivationResponse)。
type activationResponse struct {
	LicenseID string `json:"license_id"`
}

// offlineTokenResponse 离线令牌响应(release-platform OfflineTokenResponse)。
type offlineTokenResponse struct {
	LicenseID string `json:"license_id"`
	Token     struct {
		Payload   string `json:"payload"`
		KeyID     string `json:"key_id"`
		Signature string `json:"signature"`
		Algorithm string `json:"algorithm"`
	} `json:"token"`
	PublicKeyHex string `json:"public_key_hex,omitempty"`
	ExpiresAt    *string `json:"expires_at,omitempty"`
}

// apiError release-platform 统一错误体。
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"trace_id"`
}

// Exchange 完整激活:① POST /v1/activations 兑码 → ② POST /v1/licenses/{id}/offline-token
// 取离线令牌。返回 claims 与令牌原文(供本地落盘)。
func (c *APIClient) Exchange(ctx context.Context, activationCode, productID, deviceID, fingerprint string) (*Claims, []byte, error) {
	licID, err := c.activate(ctx, activationRequest{
		ActivationCode:  activationCode,
		ProductID:       productID,
		DeviceID:        deviceID,
		FingerprintHash: fingerprint,
	})
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
func (c *APIClient) activate(ctx context.Context, req activationRequest) (string, error) {
	var out activationResponse
	if err := c.do(ctx, http.MethodPost, "/v1/activations", req, &out); err != nil {
		return "", err
	}
	if out.LicenseID == "" {
		return "", fmt.Errorf("license: activation returned empty license_id")
	}
	return out.LicenseID, nil
}

// offlineToken POST /v1/licenses/{id}/offline-token。
func (c *APIClient) offlineToken(ctx context.Context, licenseID, deviceID, fingerprint string) (offlineTokenResponse, error) {
	var out offlineTokenResponse
	body := map[string]string{"device_id": deviceID, "fingerprint_hash": fingerprint}
	path := fmt.Sprintf("/v1/licenses/%s/offline-token", licenseID)
	if err := c.do(ctx, http.MethodPost, path, body, &out); err != nil {
		return out, err
	}
	if out.Token.Payload == "" || out.Token.Signature == "" {
		return out, fmt.Errorf("license: offline token incomplete")
	}
	return out, nil
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
		var ae apiError
		_ = json.Unmarshal(raw, &ae)
		return fmt.Errorf("license: %s %s -> %d %s (%s)", method, path, resp.StatusCode, ae.Message, ae.TraceID)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("license: decode %s: %w", path, err)
		}
	}
	return nil
}