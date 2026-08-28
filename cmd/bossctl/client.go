package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	// identityName --as 启用的身份档案名;401 时用于给出档案过期的修复提示。
	identityName string
}

// http 客户端:JSON 调用 60s;multipart 上传(APK 可达 32MB)放宽到 15min。
var (
	jsonClient   = &http.Client{Timeout: 60 * time.Second}
	uploadClient = &http.Client{Timeout: 15 * time.Minute}
)

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

// AuthError 认证失败(信封 code=401):main 用 errors.As 识别后给出修复提示
// (--as 身份提示档案重存,否则提示登录),取代旧实现对错误文本做字符串匹配。
type AuthError struct {
	Context string
	Code    int
	Msg     string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("%s失败: code=%d msg=%s", e.Context, e.Code, e.Msg)
}

// errBiz 业务失败(code != 0)统一转 error:主程序打印到 stderr 并以退出码 1 结束,
// CI/脚本可据此感知失败(此前 call/upload 业务失败退出码 0,流水线误判成功)。
// 401 单独包成 AuthError 供调用方识别。
func (r *apiResp) errBiz(context string) error {
	if r.Code == 401 {
		return &AuthError{Context: context, Code: r.Code, Msg: r.Msg}
	}
	return fmt.Errorf("%s失败: code=%d msg=%s", context, r.Code, r.Msg)
}

// do 发送 HTTP 请求并解析响应信封。
// data 为 nil 时跳过 body;query 可空。
func (c *CLI) do(method, path string, data any, query map[string]string) (*apiResp, error) {
	req, err := c.newRequest(method, path, data, query)
	if err != nil {
		return nil, err
	}

	resp, err := jsonClient.Do(req)
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

// newRequest 构造带认证头与 Content-Type 的请求。
func (c *CLI) newRequest(method, path string, data any, query map[string]string) (*http.Request, error) {
	full := c.cfg.Server + path
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
	if name, value := c.authHeaderValue(); name != "" {
		req.Header.Set(name, value)
	}
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// encodeQuery 编码查询参数(值经 url.QueryEscape,中文/空格/& 不再破坏 URL)。
func encodeQuery(q map[string]string) string {
	if len(q) == 0 {
		return ""
	}
	vals := url.Values{}
	for k, v := range q {
		vals.Set(k, v)
	}
	return vals.Encode()
}

// decodeEnvelope 解析统一响应信封 {code,msg,data}。
func decodeEnvelope(body []byte) (*apiResp, error) {
	var ar apiResp
	if err := json.Unmarshal(body, &ar); err != nil {
		return nil, err
	}
	return &ar, nil
}

// printJSON 格式化输出 JSON。
func printJSON(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

// queryFromArgs 从命令行参数解析 --query k=v 和 --data JSON。
// --data 支持 @file 语法从文件读大载荷;解析失败的 JSON 按原样字符串发送。
func queryFromArgs(args []string) (data any, query map[string]string, positional []string, err error) {
	query = make(map[string]string)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--data":
			if i+1 >= len(args) {
				return nil, nil, nil, fmt.Errorf("--data 缺少值")
			}
			v, e := loadDataArg(args[i+1])
			if e != nil {
				return nil, nil, nil, e
			}
			data = v
			i++
		case "--query":
			if i+1 >= len(args) {
				return nil, nil, nil, fmt.Errorf("--query 缺少值(应为 k=v)")
			}
			if k, v, ok := splitKV(args[i+1]); ok {
				query[k] = v
			}
			i++
		default:
			positional = append(positional, args[i])
		}
	}
	return data, query, positional, nil
}

// loadDataArg 解析 --data 值:@file 读文件,否则按 JSON 字面量(非法 JSON 原样字符串)。
func loadDataArg(arg string) (any, error) {
	if !strings.HasPrefix(arg, "@") {
		var v any
		if err := json.Unmarshal([]byte(arg), &v); err == nil {
			return v, nil
		}
		return arg, nil
	}
	file := strings.TrimPrefix(arg, "@")
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("读取 --data 文件: %w", err)
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("--data @%s 不是合法 JSON: %v", file, err)
	}
	return v, nil
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

// isJSON 检查字节切片是否以 JSON 对象或数组开头。
func isJSON(b []byte) bool {
	b = bytes.TrimSpace(b)
	return len(b) > 0 && (b[0] == '{' || b[0] == '[')
}
