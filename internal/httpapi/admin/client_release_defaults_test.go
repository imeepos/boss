package adminapi

// 回归:POST /client-releases 缺省 minSupportedCode 时默认=versionCode(102 冒烟发现
// 缺省 0 触发 validate 失败 50000)。

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/apprelease"
	"github.com/ymm-001/boss/internal/domain/attachment"
)

type memReleaseStore struct{ created *apprelease.Release }

// fakeObjStore 测试替身:Put 回假 key,Open 不可用(本用例不走下载)。
type fakeObjStore struct{}

func (fakeObjStore) Put(_ context.Context, _ attachment.MinIOConfig, _ io.Reader, _ int64, _, _ string) (string, error) {
	return "apk/test-key", nil
}
func (fakeObjStore) Open(_ context.Context, _ attachment.MinIOConfig, key string) (io.ReadCloser, error) {
	return nil, apprelease.ErrNotFound
}

func (m *memReleaseStore) Create(_ context.Context, r *apprelease.Release) error {
	r.ID = 1
	m.created = r
	return nil
}
func (m *memReleaseStore) Get(_ context.Context, id int64) (*apprelease.Release, error) {
	return nil, apprelease.ErrNotFound
}
func (m *memReleaseStore) List(_ context.Context, app string) ([]apprelease.Release, error) {
	return nil, nil
}
func (m *memReleaseStore) Update(_ context.Context, r *apprelease.Release) error { return nil }
func (m *memReleaseStore) Actives(_ context.Context, app, platform string) ([]apprelease.Release, error) {
	return nil, nil
}

func TestClientReleaseCreateDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	st := &memReleaseStore{}
	r := gin.New()
	// 只挂 create 一个 handler(不整 Register,避免拉全量依赖)。
	g := r.Group("/api/admin/v1")
	g.POST("/client-releases", clientReleaseCreate(&app.Application{
		AppRelease: &apprelease.Service{St: st, Obj: fakeObjStore{}},
	}))

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "app.apk")
	fw.Write([]byte("fake-apk-bytes"))
	_ = mw.WriteField("app", "user")
	_ = mw.WriteField("version", "0.2.0")
	_ = mw.WriteField("versionCode", "2")
	_ = mw.WriteField("notes", "smoke")
	_ = mw.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/client-releases", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var env struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.Code != 0 {
		t.Fatalf("envelope code=%d body=%s", env.Code, w.Body.String())
	}
	if st.created == nil || st.created.MinSupportedCode != 2 || st.created.Status != apprelease.StatusDraft {
		t.Fatalf("defaults wrong: %+v", st.created)
	}
}
