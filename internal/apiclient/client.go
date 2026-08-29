// Package apiclient 提供面向 BOSS 三端 REST API 的精简客户端:
// 统一响应信封 {code,msg,data} 解析 + X-API-Key 认证 + 查询串编码。
// cmd/bossmcp(MCP server)使用;与 cmd/bossctl 的 CLI 客户端相互独立。
package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultServer 缺省后端地址(102 部署环境,与 bossctl 缺省一致)。
const DefaultServer = "http://192.168.0.102:28080"

// Client 单主体 REST 客户端:Server 与认证凭据一一绑定。
// APIKey 为该主体专属 key(customer/worker/account 三表互不通用)。
type Client struct {
	Server string
	APIKey string
	HTTP   *http.Client
}

// Envelope 统一响应信封。
type Envelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// Result 一次调用的完整结果:HTTP 状态 + 原始响应体 + 可解析时的信封。
type Result struct {
	StatusCode  int
	ContentType string
	Body        []byte
	Envelope    *Envelope // 响应为 JSON 信封时非 nil
}

// maxErrBody 非 JSON/错误响应体截断长度,防超大载荷刷屏。
const maxErrBody = 2048

// New 构造客户端;server 为空时取 DefaultServer。
func New(server, apiKey string) *Client {
	if server == "" {
		server = DefaultServer
	}
	return &Client{
		Server: strings.TrimRight(server, "/"),
		APIKey: apiKey,
		HTTP:   &http.Client{Timeout: 60 * time.Second},
	}
}

// Call 发送请求并返回结果。data 非 nil 时 JSON 编码为 body;
// query 值经 URL 编码(中文/空格/& 安全)。
// transport 失败才返回 error;HTTP 4xx/5xx 与业务 code!=0 均在 Result 里由调用方判定。
func (c *Client) Call(method, path string, data any, query map[string]string) (*Result, error) {
	req, err := c.newRequest(method, path, data, query)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	r := &Result{
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        body,
	}
	if isJSONBody(body, r.ContentType) {
		var env Envelope
		if json.Unmarshal(body, &env) == nil && (env.Code != 0 || env.Msg != "" || string(env.Data) != "") {
			r.Envelope = &env
		}
	}
	return r, nil
}

// newRequest 构造带 X-API-Key 头与 Content-Type 的请求;path 须为服务端绝对路径。
func (c *Client) newRequest(method, path string, data any, query map[string]string) (*http.Request, error) {
	full := c.Server + path
	if len(query) > 0 {
		full += "?" + encodeQuery(query)
	}
	var body io.Reader
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, full, body)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// encodeQuery 编码查询参数。
func encodeQuery(q map[string]string) string {
	vals := url.Values{}
	for k, v := range q {
		vals.Set(k, v)
	}
	return vals.Encode()
}

// isJSONBody 判定响应体是否按 JSON 解析:content-type 声明或首字节嗅探任一命中。
// 嗅探兜底覆盖显式 WriteHeader 后 Content-Type 缺省 text/plain 的响应;误判由解析失败兜底。
func isJSONBody(body []byte, contentType string) bool {
	if strings.Contains(strings.ToLower(contentType), "json") {
		return true
	}
	b := bytes.TrimSpace(body)
	return len(b) > 0 && (b[0] == '{' || b[0] == '[')
}

// Truncate 截断响应体用于错误展示。
func Truncate(b []byte) string {
	s := string(b)
	if len(s) > maxErrBody {
		return s[:maxErrBody] + "...(truncated)"
	}
	return s
}
