package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// JSON-RPC 2.0 标准错误码(MCP 复用)。
const (
	codeParse          = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInternal       = -32603
)

// request 单条 JSON-RPC 请求/通知。
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// rpcError JSON-RPC error 对象。
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// maxLine 单条消息上限(MCP stdio 每行一条;16MB 覆盖全部文本型工具调用)。
const maxLine = 16 * 1024 * 1024

// serve 在 r 上逐行读 JSON-RPC 消息并处理,响应写 w(每行一条);EOF 正常返回。
// stdout 是协议通道:任何日志必须走 stderr(logf),否则破坏协议流。
func serve(r io.Reader, w io.Writer, s *server) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), maxLine)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		if resp := dispatch(line, s); resp != nil {
			if _, err := w.Write(append(resp, '\n')); err != nil {
				return err
			}
		}
	}
	return sc.Err()
}

// dispatch 处理单条消息;通知(无 id)返回 nil 不回包。
func dispatch(line []byte, s *server) []byte {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		return marshalResponse(nil, &rpcError{Code: codeParse, Message: "parse error"})
	}
	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		return marshalResponse(req.ID, &rpcError{Code: codeInvalidRequest, Message: "jsonrpc must be \"2.0\""})
	}
	if len(req.ID) == 0 || string(req.ID) == "null" {
		return nil // notifications/initialized、notifications/cancelled 等一律静默
	}
	switch req.Method {
	case "initialize":
		return marshalResponse(req.ID, s.resultInitialize(req.Params))
	case "ping":
		return marshalResponse(req.ID, map[string]any{})
	case "tools/list":
		return marshalResponse(req.ID, map[string]any{"tools": toolDefs()})
	case "tools/call":
		return marshalResponse(req.ID, s.resultCallTool(req.Params))
	default:
		return marshalResponse(req.ID, &rpcError{
			Code: codeMethodNotFound, Message: fmt.Sprintf("method not found: %s", req.Method),
		})
	}
}

// resultInitialize 回应 initialize:回显客户端协议版本,声明 tools 能力。
func (s *server) resultInitialize(params json.RawMessage) map[string]any {
	var p struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	_ = json.Unmarshal(params, &p)
	if p.ProtocolVersion == "" {
		p.ProtocolVersion = "2024-11-05"
	}
	return map[string]any{
		"protocolVersion": p.ProtocolVersion,
		"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
		"serverInfo":      map[string]any{"name": "bossmcp", "version": s.Version},
	}
}

// marshalResponse 序列化单行响应;v 为 *rpcError 时输出 error,否则输出 result。
func marshalResponse(id json.RawMessage, v any) []byte {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	payload := map[string]any{"jsonrpc": "2.0", "id": id}
	if rerr, ok := v.(*rpcError); ok {
		payload["error"] = rerr
	} else {
		payload["result"] = v
	}
	b, err := json.Marshal(payload)
	if err != nil {
		b, _ = json.Marshal(map[string]any{
			"jsonrpc": "2.0", "id": id,
			"error": &rpcError{Code: codeInternal, Message: err.Error()},
		})
	}
	return b
}
