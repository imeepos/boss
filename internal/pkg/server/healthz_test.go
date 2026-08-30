package server

// 回归:healthz 必须自报 commit 字段(版本自证——部署后机器核验"上没上线",
// 告别人工 curl bundle 指纹;见 docs/ops/runbook 复验口诀的脚本化替代)。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ymm-001/boss/internal/pkg/buildinfo"
)

func TestHealthzReportsCommit(t *testing.T) {
	r := New(Config{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("healthz status=%d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("healthz body not json: %v", err)
	}
	commit, _ := body["commit"].(string)
	if commit == "" {
		t.Fatalf("healthz missing commit field: %s", w.Body.String())
	}
	if body["status"] != "ok" {
		t.Fatalf("healthz status=%v", body["status"])
	}
}

// 容器构建上下文无 .git,流水线经 ldflags 注入 buildinfo.Commit,
// 必须优先于 buildvcs(其缺省缺值),并截短到 7 位。
func TestBuildCommitLdflagsPriority(t *testing.T) {
	old := buildinfo.Commit
	defer func() { buildinfo.Commit = old }()
	buildinfo.Commit = "856513e8deadbeefdeadbeefdeadbeefdeadbeef"
	if got := buildCommit(); got != "856513e" {
		t.Fatalf("buildCommit=%q, want 856513e", got)
	}
	buildinfo.Commit = "   "
	if got := buildCommit(); got == "" {
		t.Fatal("空白注入应回退而非产出空串")
	}
}
