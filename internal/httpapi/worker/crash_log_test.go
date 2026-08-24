package workerapi

// 契约:POST /client/crash(师傅端,workerJWT 鉴权);App 启动补传本地崩溃留痕。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	crashdomain "github.com/ymm-001/boss/internal/domain/crash"
	"github.com/ymm-001/boss/internal/domain/portal"
)

type memCrashStore struct {
	mu   sync.Mutex
	logs []crashdomain.Log
}

func (m *memCrashStore) Insert(_ context.Context, l crashdomain.Log) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l.Log == "" {
		return errors.New("empty log")
	}
	m.logs = append(m.logs, l)
	return nil
}

func (m *memCrashStore) ListRecent(_ context.Context, limit int) ([]crashdomain.Log, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit > len(m.logs) || limit <= 0 {
		limit = len(m.logs)
	}
	out := make([]crashdomain.Log, limit)
	for i := 0; i < limit; i++ {
		out[i] = m.logs[len(m.logs)-1-i]
	}
	return out, nil
}

func newCrashRouter(t *testing.T) (*gin.Engine, *memCrashStore) {
	t.Helper()
	t.Setenv("BOSS_JWT_SECRET", "portal-test-secret")
	gin.SetMode(gin.TestMode)
	store := &memCrashStore{}
	r := gin.New()
	Register(r, &app.Application{Portal: portal.NewMemory(), CrashLogs: store}, newWorkerJWTManager())
	return r, store
}

func TestWorkerCrashUpload(t *testing.T) {
	r, store := newCrashRouter(t)
	tok, err := signWorkerToken(9, "王师傅")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	body := `{"app":"boss-worker/0.1.0","log":"thread: main\nstack: RuntimeException boom"}`
	req := httptest.NewRequest(http.MethodPost, "/api/worker/v1/client/crash", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Fatalf("code=%d body=%s", resp.Code, w.Body.String())
	}
	logs, _ := store.ListRecent(context.Background(), 10)
	if len(logs) != 1 || logs[0].SubjectID != 9 || logs[0].App != "boss-worker/0.1.0" {
		t.Fatalf("stored=%+v", logs)
	}
}

func TestWorkerCrashUploadNoToken(t *testing.T) {
	r, _ := newCrashRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/worker/v1/client/crash",
		strings.NewReader(`{"log":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusOK {
		t.Fatalf("unauthorized request should not be 200: %s", w.Body.String())
	}
}

func TestCrashClip(t *testing.T) {
	long := strings.Repeat("a", crashdomain.MaxLogBytes+100)
	got := crashdomain.Clip(long)
	if len(got) > crashdomain.MaxLogBytes+32 {
		t.Fatalf("clip len=%d", len(got))
	}
	if !strings.HasSuffix(got, strings.Repeat("a", crashdomain.MaxLogBytes)) {
		t.Fatalf("clip should keep tail")
	}
}
