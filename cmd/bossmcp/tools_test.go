package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ymm-001/boss/internal/apiclient"
)

// recording 捕获后端收到的请求并回放统一信封。
type recording struct {
	Method, Path, RawQuery, Key, Body string
}

func backendWith(t *testing.T, respCode int, envelope string) (*httptest.Server, *recording) {
	t.Helper()
	rec := &recording{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		rec.Method, rec.Path, rec.RawQuery = r.Method, r.URL.Path, r.URL.RawQuery
		rec.Key, rec.Body = r.Header.Get("X-API-Key"), string(b)
		w.WriteHeader(respCode)
		_, _ = w.Write([]byte(envelope))
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

// serverWithKey 装配两端指向同一 backend;apiKey 为空串模拟未配置 key。
func serverWithKey(backend string, userKey, workerKey string) *server {
	s := newServer(func(string) string { return "" })
	s.Portals["user"] = &portalCfg{Name: "user", Prefix: "/api/user/v1",
		Client: apiclient.New(backend, userKey), EnvKeys: []string{"BOSS_USER_API_KEY"}}
	s.Portals["worker"] = &portalCfg{Name: "worker", Prefix: "/api/worker/v1",
		Client: apiclient.New(backend, workerKey), EnvKeys: []string{"BOSS_WORKER_API_KEY"}}
	return s
}

// callTool 走 resultCallTool 完整入口,返回 (text, isError)。
func callTool(t *testing.T, s *server, name, args string) (string, bool) {
	t.Helper()
	params, _ := json.Marshal(map[string]any{"name": name, "arguments": json.RawMessage(args)})
	res := s.resultCallTool(params)
	blocks, _ := res["content"].([]content)
	if len(blocks) == 0 {
		t.Fatalf("content missing: %v", res)
	}
	isErr, _ := res["isError"].(bool)
	return blocks[0].Text, isErr
}

func TestToolCallPrefixesPathAndSendsPortalKey(t *testing.T) {
	srv, rec := backendWith(t, 200, `{"code":0,"msg":"ok","data":{"orderNo":"ORD-1"}}`)
	s := serverWithKey(srv.URL, "boss_user_key", "boss_worker_key")

	text, isErr := callTool(t, s, "boss_call",
		`{"portal":"user","method":"post","path":"/orders","body":{"offerId":1},"query":{"page":"2"}}`)
	if isErr {
		t.Fatalf("unexpected error: %s", text)
	}
	if rec.Path != "/api/user/v1/orders" || rec.Key != "boss_user_key" {
		t.Errorf("path=%s key=%s", rec.Path, rec.Key)
	}
	if rec.Method != "POST" || rec.Body != `{"offerId":1}` || rec.RawQuery != "page=2" {
		t.Errorf("method=%s body=%s query=%s", rec.Method, rec.Body, rec.RawQuery)
	}
	if !strings.Contains(text, "ORD-1") {
		t.Errorf("text = %s", text)
	}
}

func TestToolCallRejectsCrossPortalPath(t *testing.T) {
	srv, rec := backendWith(t, 200, `{"code":0,"msg":"ok","data":null}`)
	s := serverWithKey(srv.URL, "boss_user_key", "")

	text, isErr := callTool(t, s, "boss_call", `{"portal":"user","method":"GET","path":"/api/worker/v1/home"}`)
	if !isErr {
		t.Fatalf("expected cross-portal rejection, got: %s", text)
	}
	if !strings.Contains(text, "不属于 user 端") {
		t.Errorf("text = %s", text)
	}
	if rec.Key != "" {
		t.Errorf("key must not be sent to wrong portal, got %s", rec.Key)
	}
}

func TestToolCallMissingKeyFailsFast(t *testing.T) {
	srv, _ := backendWith(t, 200, `{"code":0,"msg":"ok","data":null}`)
	s := serverWithKey(srv.URL, "", "boss_worker_key")

	text, isErr := callTool(t, s, "boss_call", `{"portal":"user","method":"GET","path":"/orders"}`)
	if !isErr || !strings.Contains(text, "BOSS_USER_API_KEY") {
		t.Errorf("isErr=%v text=%s", isErr, text)
	}
}

func TestToolCallBusinessErrorIsErrorResult(t *testing.T) {
	srv, _ := backendWith(t, 403, `{"code":40300,"msg":"无权限"}`)
	s := serverWithKey(srv.URL, "k", "k")

	text, isErr := callTool(t, s, "boss_call", `{"portal":"worker","method":"GET","path":"/tickets"}`)
	if !isErr || !strings.Contains(text, "40300") || !strings.Contains(text, "无权限") {
		t.Errorf("isErr=%v text=%s", isErr, text)
	}
}

func TestToolCallUnknownPortalAndMethod(t *testing.T) {
	srv, _ := backendWith(t, 200, `{"code":0,"msg":"ok","data":null}`)
	s := serverWithKey(srv.URL, "k", "k")

	if text, isErr := callTool(t, s, "boss_call", `{"portal":"admin","method":"GET","path":"/orders"}`); !isErr || !strings.Contains(text, "未知端") {
		t.Errorf("admin rejected: isErr=%v text=%s", isErr, text)
	}
	if text, isErr := callTool(t, s, "boss_call", `{"portal":"user","method":"CONNECT","path":"/orders"}`); !isErr || !strings.Contains(text, "不支持的方法") {
		t.Errorf("bad method rejected: isErr=%v text=%s", isErr, text)
	}
}

func TestToolWhoamiHitsProfile(t *testing.T) {
	srv, rec := backendWith(t, 200, `{"code":0,"msg":"ok","data":{"realName":"张三"}}`)
	s := serverWithKey(srv.URL, "boss_u", "boss_w")

	if _, isErr := callTool(t, s, "boss_whoami", `{"portal":"worker"}`); isErr {
		t.Fatal("unexpected error")
	}
	if rec.Path != "/api/worker/v1/profile" || rec.Key != "boss_w" {
		t.Errorf("path=%s key=%s", rec.Path, rec.Key)
	}
}

func TestToolRoutesFilter(t *testing.T) {
	s := serverWithKey("http://unused", "k", "k")

	text, isErr := callTool(t, s, "boss_routes", `{"portal":"user","filter":"tickets"}`)
	if isErr {
		t.Fatalf("unexpected error: %s", text)
	}
	if !strings.Contains(text, "/api/user/v1") || !strings.Contains(text, "共 ") {
		t.Errorf("text = %s", text)
	}
	for _, ln := range strings.Split(text, "\n") {
		if strings.HasPrefix(ln, "GET") || strings.HasPrefix(ln, "POST") {
			if !strings.Contains(strings.ToLower(ln), "ticket") {
				t.Errorf("filter leaked: %s", ln)
			}
		}
	}
}

func TestResolvePath(t *testing.T) {
	p := &portalCfg{Name: "user", Prefix: "/api/user/v1"}
	cases := []struct {
		in, want string
	}{
		{"/orders", "/api/user/v1/orders"},
		{"orders", "/api/user/v1/orders"},
		{"/api/user/v1/orders", "/api/user/v1/orders"},
		{"/profile/security/password", "/api/user/v1/profile/security/password"},
	}
	for _, c := range cases {
		got, err := resolvePath(p, c.in)
		if err != nil || got != c.want {
			t.Errorf("resolvePath(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	if _, err := resolvePath(p, "  "); err == nil {
		t.Error("empty path must fail")
	}
}

func TestNormalizeMethod(t *testing.T) {
	if m, _ := normalizeMethod(" get "); m != "GET" {
		t.Errorf("normalizeMethod = %q", m)
	}
	if _, err := normalizeMethod("TRACE"); err == nil {
		t.Error("TRACE must be rejected")
	}
}

func TestNewServerKeyFallback(t *testing.T) {
	env := map[string]string{"BOSS_USER_API_KEY": "u1", "BOSS_API_KEY": "fallback", "BOSS_SERVER": "http://srv"}
	s := newServer(func(k string) string { return env[k] })
	if s.Portals["user"].Client.APIKey != "u1" || s.Portals["user"].EnvKeys[0] != "BOSS_USER_API_KEY" {
		t.Errorf("user = %+v", s.Portals["user"])
	}
	if s.Portals["worker"].Client.APIKey != "fallback" || s.Portals["worker"].EnvKeys[0] != "BOSS_API_KEY" {
		t.Errorf("worker fallback = %+v", s.Portals["worker"])
	}
	if s.Portals["user"].Client.Server != "http://srv" {
		t.Errorf("server = %q", s.Portals["user"].Client.Server)
	}

	missing := newServer(func(string) string { return "" })
	if missing.Portals["worker"].Client.APIKey != "" || missing.Portals["worker"].EnvKeys[0] != "BOSS_WORKER_API_KEY" {
		t.Errorf("missing = %+v", missing.Portals["worker"])
	}
	if err := missing.Portals["worker"].requireKey(); err == nil {
		t.Error("requireKey must fail without key")
	}
}
