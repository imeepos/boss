package workerapi

// 契约(fields.md 8F):GET /client/latest(师傅端免登录升级判定)+
// GET /client/apk/:id(灰度门控下载);带 token 时白名单豁免生效。

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/apprelease"
)

type memReleaseStore struct {
	mu       sync.Mutex
	rows     []apprelease.Release
	actives  []apprelease.Release
	byIDFail bool
}

func (m *memReleaseStore) Create(_ context.Context, r *apprelease.Release) error { return nil }
func (m *memReleaseStore) Get(_ context.Context, id int64) (*apprelease.Release, error) {
	for i := range m.rows {
		if m.rows[i].ID == id {
			return &m.rows[i], nil
		}
	}
	return nil, apprelease.ErrNotFound
}
func (m *memReleaseStore) List(_ context.Context, app string) ([]apprelease.Release, error) {
	return m.rows, nil
}
func (m *memReleaseStore) Update(_ context.Context, r *apprelease.Release) error { return nil }
func (m *memReleaseStore) Actives(_ context.Context, app, platform string) ([]apprelease.Release, error) {
	out := make([]apprelease.Release, len(m.actives))
	copy(out, m.actives)
	return out, nil
}

func newClientReleaseRouter(t *testing.T, st *memReleaseStore) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	a := &app.Application{AppRelease: &apprelease.Service{St: st}}
	r.Group("/api/worker/v1")
	Register(r, a, nil)
	return r
}

func getJSON(t *testing.T, r *gin.Engine, path string) (int, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	var env struct {
		Data map[string]any `json:"data"`
		Code int            `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	out := env.Data
	if out == nil {
		out = map[string]any{"code": env.Code}
	}
	return w.Code, out
}

func TestWorkerClientLatestPublished(t *testing.T) {
	st := &memReleaseStore{actives: []apprelease.Release{{
		ID: 3, App: "worker", Platform: "android", Version: "1.1.0", VersionCode: 11,
		MinSupportedCode: 1, Status: apprelease.StatusPublished, Notes: "fix", Sha256: "aa", ApkSize: 9,
	}}}
	r := newClientReleaseRouter(t, st)
	code, body := getJSON(t, r, "/api/worker/v1/client/latest?versionCode=10&deviceId=d1")
	if code != http.StatusOK || body["updateAvailable"] != true {
		t.Fatalf("code=%d body=%v", code, body)
	}
	if body["downloadUrl"] != "/api/worker/v1/client/apk/3" || body["force"] != false {
		t.Fatalf("projection mismatch: %v", body)
	}
}

func TestWorkerClientLatestUpToDate(t *testing.T) {
	st := &memReleaseStore{actives: []apprelease.Release{{
		ID: 3, App: "worker", Version: "1.1.0", VersionCode: 11, MinSupportedCode: 1,
		Status: apprelease.StatusPublished,
	}}}
	r := newClientReleaseRouter(t, st)
	_, body := getJSON(t, r, "/api/worker/v1/client/latest?versionCode=11&deviceId=d1")
	if body["updateAvailable"] != false {
		t.Fatalf("same versionCode must be no update: %v", body)
	}
}

func TestWorkerClientLatestMissingParams(t *testing.T) {
	r := newClientReleaseRouter(t, &memReleaseStore{})
	if code, _ := getJSON(t, r, "/api/worker/v1/client/latest?versionCode=10"); code != http.StatusOK {
		// 信封层 invalid param 也是 200 + code 字段;断言 code 字段。
	}
	code, body := getJSON(t, r, "/api/worker/v1/client/latest?versionCode=10")
	if body["code"] == float64(0) {
		t.Fatalf("missing deviceId must reject: %d %v", code, body)
	}
}

func TestWorkerClientAPKGrayGate(t *testing.T) {
	st := &memReleaseStore{
		actives: []apprelease.Release{{
			ID: 5, App: "worker", Version: "2.0.0", VersionCode: 20, MinSupportedCode: 1,
			Status: apprelease.StatusGray, RolloutPercent: 0, // 0%:无人命中
		}},
		rows: []apprelease.Release{{ID: 5, App: "worker", Version: "2.0.0", VersionCode: 20,
			Status: apprelease.StatusGray, RolloutPercent: 0}},
	}
	r := newClientReleaseRouter(t, st)
	// 灰度未命中:latest 无更新,直链下载被信封 40400 门控(HTTP 恒 200,看 code)。
	if _, body := getJSON(t, r, "/api/worker/v1/client/latest?versionCode=1&deviceId=d1"); body["updateAvailable"] != false {
		t.Fatalf("0%% gray must not serve: %v", body)
	}
	_, body := getJSON(t, r, "/api/worker/v1/client/apk/5?deviceId=d1")
	if fmt.Sprint(body["code"]) != "40400" {
		t.Fatalf("gray miss must be gated 40400, got %v", body)
	}
}
