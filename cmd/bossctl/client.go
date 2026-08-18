package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// config CLI 全局配置。
type config struct {
	Server string
	APIKey string
	JWT    string
}

// CLI 持有配置和 HTTP 客户端。
type CLI struct {
	cfg *config
}

// authHeaderValue 返回认证请求头名称和值。
// API key: X-API-Key 头; JWT: Authorization: Bearer <token>。
func (c *CLI) authHeaderValue() (name, value string) {
	if c.cfg.APIKey != "" {
		return "X-API-Key", c.cfg.APIKey
	}
	if c.cfg.JWT != "" {
		return "Authorization", "Bearer " + c.cfg.JWT
	}
	// 尝试从缓存文件读取 JWT
	tok, _ := loadToken()
	if tok != "" {
		return "Authorization", "Bearer " + tok
	}
	return "", ""
}

// tokenFile 返回 JWT 缓存文件路径。
func tokenFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".bossctl", "token")
}

// saveToken 保存 JWT 到缓存文件。
func saveToken(token string) error {
	dir := filepath.Dir(tokenFile())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(tokenFile(), []byte(token), 0600)
}

// loadToken 从缓存文件读取 JWT。
func loadToken() (string, error) {
	b, err := os.ReadFile(tokenFile())
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(b)), nil
}

// clearToken 清除缓存的 JWT。
func clearToken() {
	os.Remove(tokenFile())
}

// apiResp 统一响应信封。
type apiResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// do 发送 HTTP 请求并解析响应信封。
// data 为 nil 时跳过 body;query 可空。
func (c *CLI) do(method, path string, data any, query map[string]string) (*apiResp, error) {
	url := c.cfg.Server + path
	if len(query) > 0 {
		url += "?" + encodeQuery(query)
	}

	var body io.Reader
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	// 设置认证头
	if name, value := c.authHeaderValue(); name != "" {
		req.Header.Set(name, value)
	}
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// 非 JSON 响应(如 404 HTML 页面)返回清晰错误
	if resp.StatusCode != http.StatusOK && !isJSON(respBody) {
		return nil, fmt.Errorf("服务器返回 %d (非 JSON 响应,可能路径不存在)", resp.StatusCode)
	}

	var ar apiResp
	if err := json.Unmarshal(respBody, &ar); err != nil {
		// 非 JSON = 路由不存在(如 Gin 的 404 HTML)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("404: 路径不存在(检查 bossctl routes 或 --server 地址)")
		}
		return nil, fmt.Errorf("parse response: %s: %w", string(respBody), err)
	}
	return &ar, nil
}

// encodeQuery 编码查询参数。
func encodeQuery(q map[string]string) string {
	if len(q) == 0 {
		return ""
	}
	var buf bytes.Buffer
	for k, v := range q {
		if buf.Len() > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(v)
	}
	return buf.String()
}

// printJSON 格式化输出 JSON。
func printJSON(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

// queryFromArgs 从命令行参数解析 --query k=v 和 --data JSON。
func queryFromArgs(args []string) (data any, query map[string]string, positional []string) {
	query = make(map[string]string)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--data":
			if i+1 < len(args) {
				var v any
				if err := json.Unmarshal([]byte(args[i+1]), &v); err == nil {
					data = v
				} else {
					data = args[i+1]
				}
				i++
			}
		case "--query":
			if i+1 < len(args) {
				if k, v, ok := splitKV(args[i+1]); ok {
					query[k] = v
				}
				i++
			}
		default:
			positional = append(positional, args[i])
		}
	}
	return
}

// splitKV 分割 k=v 格式。
func splitKV(s string) (k, v string, ok bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}