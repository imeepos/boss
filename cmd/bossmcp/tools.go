package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ymm-001/boss/internal/apiclient"
	"github.com/ymm-001/boss/internal/apiroutes"
)

// toolDef MCP 工具定义。
type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// toolDefs MCP tools/list 载荷,与 resultCallTool 的分支一一对应。
func toolDefs() []toolDef {
	return []toolDef{
		{
			Name: "boss_routes",
			Description: "列出 BOSS 平台指定端的 API 路由目录(user=用户端/客户门户,worker=师傅端)。" +
				"先用本工具发现接口,再用 boss_call 调用;filter 可按子串过滤。",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"portal": map[string]any{"type": "string", "enum": []string{"user", "worker"},
						"description": "端: user=用户端(客户主体), worker=师傅端(装维主体)"},
					"filter": map[string]any{"type": "string",
						"description": "可选子串过滤,匹配方法/路径/描述"},
				},
				"required": []string{"portal"},
			},
		},
		{
			Name: "boss_call",
			Description: "调用 BOSS 平台 REST API。path 相对端前缀(如 /orders),也可传完整路径但必须属于该端" +
				"(customer/worker key 不通用,跨端路径直接拒绝)。body 为 JSON 请求体,query 为查询参数键值对。" +
				"认证按端自动注入: user 端用 BOSS_USER_API_KEY,worker 端用 BOSS_WORKER_API_KEY。",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"portal": map[string]any{"type": "string", "enum": []string{"user", "worker"}},
					"method": map[string]any{"type": "string",
						"enum": []string{"GET", "POST", "PUT", "DELETE", "PATCH"}},
					"path": map[string]any{"type": "string", "description": "接口路径,如 /orders 或 /orders/{orderNo} 的实值路径"},
					"body": map[string]any{"description": "JSON 请求体(GET/DELETE 可省略)"},
					"query": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"},
						"description": "查询参数键值对(值一律字符串)"},
				},
				"required": []string{"portal", "method", "path"},
			},
		},
		{
			Name:        "boss_whoami",
			Description: "查看当前端认证身份(user 端查 GET /profile,worker 端查 GET /profile)。开工前先确认 key 有效、主体正确。",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"portal": map[string]any{"type": "string", "enum": []string{"user", "worker"}},
				},
				"required": []string{"portal"},
			},
		},
	}
}

// toolCallParams tools/call 参数。
type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// content MCP 文本内容块。
type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// resultCallTool 分发工具调用;业务失败经 errorResult(isError=true)回传,协议层不报错。
func (s *server) resultCallTool(params json.RawMessage) map[string]any {
	var p toolCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return errorResult(fmt.Sprintf("params 解析失败: %v", err))
	}
	var (
		text string
		err  error
	)
	switch p.Name {
	case "boss_routes":
		text, err = s.toolRoutes(p.Arguments)
	case "boss_call":
		text, err = s.toolCall(p.Arguments)
	case "boss_whoami":
		text, err = s.toolWhoami(p.Arguments)
	default:
		return errorResult(fmt.Sprintf("未知工具: %s(可用 boss_routes/boss_call/boss_whoami)", p.Name))
	}
	if err != nil {
		return errorResult(err.Error())
	}
	return map[string]any{
		"content": []content{{Type: "text", Text: text}},
		"isError": false,
	}
}

// errorResult 工具执行失败:MCP 规范要求执行错误作为结果回传(isError=true)。
func errorResult(text string) map[string]any {
	return map[string]any{
		"content": []content{{Type: "text", Text: text}},
		"isError": true,
	}
}

// toolRoutes boss_routes:列端路由目录,filter 子串过滤。
func (s *server) toolRoutes(args json.RawMessage) (string, error) {
	var in struct {
		Portal string `json:"portal"`
		Filter string `json:"filter"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}
	p, err := s.portal(in.Portal)
	if err != nil {
		return "", err
	}
	dir := apiroutes.ByName(p.Name)
	var b strings.Builder
	fmt.Fprintf(&b, "[%s 端 · 前缀 %s · 认证 %s]\n", p.Name, p.Prefix, p.EnvKeys[0])
	if p.Client.APIKey == "" {
		fmt.Fprintf(&b, "[!] %s 未配置:boss_call/boss_whoami 将拒绝调用\n", p.EnvKeys[0])
	}
	n := 0
	for _, r := range dir.Routes {
		if in.Filter != "" && !routeMatch(r, in.Filter) {
			continue
		}
		fmt.Fprintf(&b, "%-6s %s    %s\n", r.Method, r.Path, r.Desc)
		n++
	}
	fmt.Fprintf(&b, "共 %d/%d 条;调用接口: boss_call(portal=%q, method, path)。", n, len(dir.Routes), p.Name)
	return b.String(), nil
}

// routeMatch 过滤命中:方法/路径/描述任一含子串(大小写不敏感)。
func routeMatch(r apiroutes.Route, filter string) bool {
	f := strings.ToLower(filter)
	return strings.Contains(strings.ToLower(r.Method), f) ||
		strings.Contains(strings.ToLower(r.Path), f) ||
		strings.Contains(strings.ToLower(r.Desc), f)
}

// toolCall boss_call:调用任意端内接口。
func (s *server) toolCall(args json.RawMessage) (string, error) {
	var in struct {
		Portal string            `json:"portal"`
		Method string            `json:"method"`
		Path   string            `json:"path"`
		Body   json.RawMessage   `json:"body"`
		Query  map[string]string `json:"query"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}
	p, err := s.portal(in.Portal)
	if err != nil {
		return "", err
	}
	if err := p.requireKey(); err != nil {
		return "", err
	}
	method, err := normalizeMethod(in.Method)
	if err != nil {
		return "", err
	}
	full, err := resolvePath(p, in.Path)
	if err != nil {
		return "", err
	}
	var data any
	if len(in.Body) > 0 && string(in.Body) != "null" {
		data = json.RawMessage(in.Body) // 原样透传,不重编码
	}
	res, err := p.Client.Call(method, full, data, in.Query)
	return renderResult(res, err)
}

// whoamiPath 各端身份查看端点(契约内端点,不新造)。
var whoamiPath = map[string]string{"user": "/profile", "worker": "/profile"}

// toolWhoami boss_whoami:查当前端认证身份。
func (s *server) toolWhoami(args json.RawMessage) (string, error) {
	var in struct {
		Portal string `json:"portal"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}
	p, err := s.portal(in.Portal)
	if err != nil {
		return "", err
	}
	if err := p.requireKey(); err != nil {
		return "", err
	}
	res, err := p.Client.Call("GET", p.Prefix+whoamiPath[p.Name], nil, nil)
	return renderResult(res, err)
}

// renderResult 把调用结果渲染为 agent 可读文本。
// 业务失败(code!=0)与 HTTP 错误以 error 返回 → isError=true;非 JSON 响应原样截断回传。
func renderResult(res *apiclient.Result, err error) (string, error) {
	if err != nil {
		return "", err
	}
	if res.Envelope != nil {
		if res.Envelope.Code != 0 || res.StatusCode >= 400 {
			return "", fmt.Errorf("HTTP %d code=%d msg=%s",
				res.StatusCode, res.Envelope.Code, res.Envelope.Msg)
		}
		var pretty bytes.Buffer
		if json.Indent(&pretty, res.Envelope.Data, "", "  ") != nil {
			return fmt.Sprintf("code=0 msg=%q data=%s", res.Envelope.Msg, string(res.Envelope.Data)), nil
		}
		return fmt.Sprintf("code=0 msg=%q\ndata:\n%s", res.Envelope.Msg, pretty.String()), nil
	}
	return fmt.Sprintf("HTTP %d %s\n%s", res.StatusCode, res.ContentType, apiclient.Truncate(res.Body)), nil
}

// normalizeMethod 白名单校验并大写化 HTTP 方法。
func normalizeMethod(m string) (string, error) {
	switch v := strings.ToUpper(strings.TrimSpace(m)); v {
	case "GET", "POST", "PUT", "DELETE", "PATCH":
		return v, nil
	default:
		return "", fmt.Errorf("不支持的方法 %q(GET/POST/PUT/DELETE/PATCH)", m)
	}
}

// resolvePath 规整调用路径:相对路径补端前缀;完整路径必须属于该端
// (customer/worker key 不通用,跨端路径直接拒绝,防止把 key 发错面)。
func resolvePath(p *portalCfg, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path 不能为空")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if strings.HasPrefix(path, "/api/") {
		if path == p.Prefix || strings.HasPrefix(path, p.Prefix+"/") {
			return path, nil
		}
		return "", fmt.Errorf("路径 %s 不属于 %s 端(%s):三表主体 key 不通用,禁止跨端携带", path, p.Name, p.Prefix)
	}
	return p.Prefix + path, nil
}
