package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestCallTargetGuard 位置参数不足时报用法错,不允许回到 index-out-of-range panic。
func TestCallTargetGuard(t *testing.T) {
	if _, _, err := callTarget([]string{"GET"}); err == nil {
		t.Fatal("仅 METHOD 无 PATH 应报错")
	}
	if _, _, err := callTarget(nil); err == nil {
		t.Fatal("空位置参数应报错")
	}
	m, p, err := callTarget([]string{"get", "user:/orders"})
	if err != nil {
		t.Fatalf("合法参数报错: %v", err)
	}
	if m != "GET" || p != "/api/user/v1/orders" {
		t.Errorf("got %s %s,期望 GET /api/user/v1/orders", m, p)
	}
}

// TestCallFlagsFirst flag 在前吃掉位置参数时仍能取到 METHOD/PATH(panic 回归)。
func TestCallFlagsFirst(t *testing.T) {
	_, _, positional, err := queryFromArgs([]string{"--data", "{}", "--query", "page=1", "GET", "/orders"})
	if err != nil {
		t.Fatalf("queryFromArgs: %v", err)
	}
	m, p, err := callTarget(positional)
	if err != nil {
		t.Fatalf("callTarget: %v", err)
	}
	if m != "GET" || p != "/api/admin/v1/orders" {
		t.Errorf("got %s %s,期望 GET /api/admin/v1/orders", m, p)
	}
}

// TestCallBizErrorExit 业务 code!=0 时 call 返回 error(main 据此退出码 1)。
func TestCallBizErrorExit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"code": 40400, "msg": "资源不存在"})
	}))
	defer srv.Close()
	c := &CLI{cfg: &config{Server: srv.URL, APIKey: "boss_test"}}
	if err := c.call([]string{"GET", "/orders/NOSUCH"}); err == nil {
		t.Fatal("业务失败应返回 error(退出码 1),实际 nil")
	}
}

// TestCallQueryEncoded 查询参数经 URL 编码(中文/空格/& 不破坏请求)。
func TestCallQueryEncoded(t *testing.T) {
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "success", "data": map[string]any{"items": []any{}}})
	}))
	defer srv.Close()
	c := &CLI{cfg: &config{Server: srv.URL, APIKey: "boss_test"}}
	if err := c.call([]string{"GET", "/orders", "--query", "keyword=宽带 安装&更多"}); err != nil {
		t.Fatalf("call: %v", err)
	}
	if gotRawQuery != "keyword=%E5%AE%BD%E5%B8%A6+%E5%AE%89%E8%A3%85%26%E6%9B%B4%E5%A4%9A" {
		t.Errorf("RawQuery = %q,期望中文与 & 均被转义", gotRawQuery)
	}
}

// TestCallDataFromFile --data @file 读取 JSON 载荷。
func TestCallDataFromFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "payload.json")
	if err := os.WriteFile(file, []byte(`{"name":"测试"}`), 0600); err != nil {
		t.Fatal(err)
	}

	var gotName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		gotName, _ = body["name"].(string)
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "success", "data": map[string]any{}})
	}))
	defer srv.Close()
	c := &CLI{cfg: &config{Server: srv.URL, APIKey: "boss_test"}}
	if err := c.call([]string{"POST", "/products", "--data", "@" + file}); err != nil {
		t.Fatalf("call: %v", err)
	}
	if gotName != "测试" {
		t.Errorf("body name = %q,期望 测试", gotName)
	}
}

// TestCallDataFileMissing @ 指向不存在的文件必须报错而非静默发空。
func TestCallDataFileMissing(t *testing.T) {
	if _, _, _, err := queryFromArgs([]string{"--data", "@/no/such/file.json", "POST", "/x"}); err == nil {
		t.Fatal("--data @缺失文件应报错")
	}
}

// TestParseSubject 主体标识严格校验(纯数字 ID;Sscanf 会放过 "5abc")。
func TestParseSubject(t *testing.T) {
	if typ, ref, err := parseSubject("worker/5"); err != nil || typ != "worker" || ref != 5 {
		t.Errorf("worker/5 解析错误: %v %d %v", typ, ref, err)
	}
	for _, bad := range []string{"worker/5abc", "worker/-1", "worker/", "boss/5", "worker"} {
		if _, _, err := parseSubject(bad); err == nil {
			t.Errorf("parseSubject(%q) 应报错", bad)
		}
	}
}

// TestEncodeQuery 转义行为直接断言。
func TestEncodeQuery(t *testing.T) {
	if got := encodeQuery(map[string]string{"a": "x y", "b": "1&2"}); got != "a=x+y&b=1%262" {
		t.Errorf("encodeQuery = %q", got)
	}
	if encodeQuery(nil) != "" {
		t.Error("空 query 应返回空串")
	}
}

// TestResolveServer 服务端地址优先级:显式 > BOSS_SERVER > 缺省。
func TestResolveServer(t *testing.T) {
	t.Setenv("BOSS_SERVER", "http://env:1")
	if got := resolveServer(""); got != "http://env:1" {
		t.Errorf("env 未生效: %q", got)
	}
	if got := resolveServer("http://flag:2/"); got != "http://flag:2" {
		t.Errorf("flag 应优先且去尾斜杠: %q", got)
	}
	os.Unsetenv("BOSS_SERVER")
	if got := resolveServer(""); got != defaultServer {
		t.Errorf("缺省应为 102 部署环境: %q", got)
	}
}

// TestSplitKV 基本行为。
func TestSplitKV(t *testing.T) {
	if k, v, ok := splitKV("page=2"); !ok || k != "page" || v != "2" {
		t.Errorf("splitKV(page=2) = %q %q %v", k, v, ok)
	}
	if _, _, ok := splitKV("flagonly"); ok {
		t.Error("无 = 应返回 false")
	}
}

// TestCheckAPIKey 身份档案 key 形态校验。
func TestCheckAPIKey(t *testing.T) {
	if err := checkAPIKey("x", "boss_00ff"); err != nil {
		t.Errorf("合法 key 报错: %v", err)
	}
	if err := checkAPIKey("x", "boss_00ff\n中文"); err == nil {
		t.Error("带换行/中文尾巴的 key 应报错")
	}
}

// TestLogoutClearsToken logout 调服务端后必须清掉本地缓存 JWT。
func TestLogoutClearsToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "success", "data": nil})
	}))
	defer srv.Close()

	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	tokenFile := filepath.Join(tmp, ".bossctl", "token")
	if err := os.MkdirAll(filepath.Dir(tokenFile), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tokenFile, []byte("jwt-x"), 0600); err != nil {
		t.Fatal(err)
	}

	c := &CLI{cfg: &config{Server: srv.URL}}
	if err := c.logout(); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := os.Stat(tokenFile); !os.IsNotExist(err) {
		t.Error("logout 后本地 token 文件应已删除")
	}
}

// TestRoutesCatalogNoDuplicate 单端路由目录无重复项(生成器去重回归;
// 同名路径跨端并存合法,如 GET /home 同时存在于 user/worker 端)。
func TestRoutesCatalogNoDuplicate(t *testing.T) {
	total := 0
	for _, p := range portalPrefixes {
		seen := map[string]bool{}
		for _, r := range p.Routes {
			k := r.Method + " " + r.Path
			if seen[k] {
				t.Errorf("%s 端路由目录重复: %s", p.Name, k)
			}
			seen[k] = true
		}
		total += len(p.Routes)
	}
	if total < 600 {
		t.Errorf("路由目录共 %d 条,明显少于契约规模,疑似生成漂移", total)
	}
}
