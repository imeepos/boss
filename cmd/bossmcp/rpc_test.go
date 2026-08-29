package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// runServe 喂入请求行,返回按行拆分的响应 JSON。
func runServe(t *testing.T, s *server, lines ...string) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := serve(strings.NewReader(strings.Join(lines, "\n")+"\n"), &out, s); err != nil {
		t.Fatalf("serve: %v", err)
	}
	var resps []map[string]any
	for _, ln := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if ln == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(ln), &m); err != nil {
			t.Fatalf("响应非法 JSON: %v: %s", err, ln)
		}
		resps = append(resps, m)
	}
	return resps
}

func testServer() *server {
	return newServer(func(string) string { return "" })
}

func TestServeInitializeEchoesClientVersion(t *testing.T) {
	resps := runServe(t, testServer(),
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`)
	if len(resps) != 1 {
		t.Fatalf("responses = %d", len(resps))
	}
	res, _ := resps[0]["result"].(map[string]any)
	if res == nil {
		t.Fatalf("result missing: %v", resps[0])
	}
	if res["protocolVersion"] != "2025-06-18" {
		t.Errorf("protocolVersion = %v", res["protocolVersion"])
	}
	info, _ := res["serverInfo"].(map[string]any)
	if info["name"] != "bossmcp" {
		t.Errorf("serverInfo = %v", info)
	}
	if _, ok := res["capabilities"].(map[string]any)["tools"]; !ok {
		t.Errorf("capabilities.tools missing: %v", res)
	}
}

func TestServeToolsList(t *testing.T) {
	resps := runServe(t, testServer(), `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	res, _ := resps[0]["result"].(map[string]any)
	tools, _ := res["tools"].([]any)
	if len(tools) != 3 {
		t.Fatalf("tools = %d, want 3", len(tools))
	}
	names := map[string]bool{}
	for _, tl := range tools {
		names[tl.(map[string]any)["name"].(string)] = true
	}
	for _, want := range []string{"boss_routes", "boss_call", "boss_whoami"} {
		if !names[want] {
			t.Errorf("tool %s missing", want)
		}
	}
}

func TestServeNotificationSilent(t *testing.T) {
	resps := runServe(t, testServer(),
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":9}}`)
	if len(resps) != 0 {
		t.Fatalf("notifications must not reply, got %v", resps)
	}
}

func TestServePingAndUnknownMethod(t *testing.T) {
	resps := runServe(t, testServer(),
		`{"jsonrpc":"2.0","id":3,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":4,"method":"resources/list"}`)
	if resps[0]["result"] == nil {
		t.Errorf("ping result missing: %v", resps[0])
	}
	errObj, _ := resps[1]["error"].(map[string]any)
	if errObj["code"] != float64(codeMethodNotFound) {
		t.Errorf("unknown method error = %v", resps[1])
	}
}

func TestServeParseErrorAndEmptyLines(t *testing.T) {
	resps := runServe(t, testServer(), ``, `not-json`, `{"jsonrpc":"1.0","id":9,"method":"x"}`)
	errObj, _ := resps[0]["error"].(map[string]any)
	if errObj["code"] != float64(codeParse) {
		t.Errorf("parse error = %v", resps[0])
	}
	errObj, _ = resps[1]["error"].(map[string]any)
	if errObj["code"] != float64(codeInvalidRequest) {
		t.Errorf("bad jsonrpc version = %v", resps[1])
	}
}
