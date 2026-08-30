package server

// 回归:healthz 必须自报 commit 字段(版本自证——部署后机器核验"上没上线",
// 告别人工 curl bundle 指纹;见 docs/ops/runbook 复验口诀的脚本化替代)。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
