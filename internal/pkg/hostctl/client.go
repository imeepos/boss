// Package hostctl 提供与宿主机 sidecar (cmd/hostctl) 的 HMAC 鉴权通信客户端。
package hostctl

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Client 调用宿主机 hostctl sidecar。
type Client struct {
	BaseURL string
	HMACKey string
	HTTP    *http.Client
}

// New 创建客户端。url 示例: http://172.26.0.1:39093。
func New(url, hmacKey string) *Client {
	return &Client{
		BaseURL: url,
		HMACKey: hmacKey,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

// RotateResp hostctl 轮换结果。
type RotateResp struct {
	OK        bool   `json:"ok"`
	RotatedAt string `json:"rotatedAt"`
	Duration  string `json:"duration"`
}

// RotateSecret 调用 POST /v1/minio/rotate-secret。
func (c *Client) RotateSecret(secret string) (*RotateResp, error) {
	body, _ := json.Marshal(map[string]string{"secret": secret})
	path := "/v1/minio/rotate-secret"
	req, err := http.NewRequest("POST", c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set("X-Hostctl-Timestamp", ts)
	sig := c.sign(ts, "POST", path, string(body))
	req.Header.Set("X-Hostctl-Signature", sig)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("hostctl %d: %s", resp.StatusCode, errResp.Error)
	}
	var r RotateResp
	if err := json.Unmarshal(respBody, &r); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &r, nil
}

// Healthz 调用 GET /healthz（无需鉴权）。
func (c *Client) Healthz() error {
	resp, err := c.HTTP.Get(c.BaseURL + "/healthz")
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hostctl healthz: %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) sign(ts, method, path, body string) string {
	mac := hmac.New(sha256.New, []byte(c.HMACKey))
	fmt.Fprintf(mac, "%s\n%s\n%s\n%s", ts, method, path, body)
	return hex.EncodeToString(mac.Sum(nil))
}
