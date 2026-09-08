package adminapi

// 契约:POST /storage-config/test(menu:params)——零副作用连通性自检。
// 结果 {ok,latencyMs,message}(与 auth-config 自检同形);失败原因进 message 供复制。
// 网络探测路径不做单测(需真实 MinIO);此处锁参数缺失短路分支与响应形状。

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/attachment"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeStorageUser:权限放行 + ListParams 空表(storage 测试走 Attachment.Resolve 注入)。
type fakeStorageUser struct {
	fakeUser
}

func (f *fakeStorageUser) ListParams(context.Context) ([]user.Param, error) {
	return nil, nil
}

func newStorageTestRouter(svc *attachment.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("test-secret", time.Hour)
	r := gin.New()
	Register(r, &app.Application{User: &fakeStorageUser{fakeUser: fakeUser{permOk: true}}, Attachment: svc}, mgr)
	return r
}

func TestStorageConfigTest_MissingEndpoint(t *testing.T) {
	svc := &attachment.Service{Resolve: func(context.Context) (attachment.MinIOConfig, error) {
		return attachment.MinIOConfig{}, nil
	}}
	r := newStorageTestRouter(svc)
	mgr := auth.NewManager("test-secret", time.Hour)
	token, err := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	if err != nil {
		t.Fatal(err)
	}
	w := postJSONAuth(t, r, "/api/admin/v1/storage-config/test", `{}`, token)
	var out struct {
		Data struct {
			Ok      bool   `json:"ok"`
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, w.Body.String())
	}
	if out.Data.Ok || out.Data.Message == "" {
		t.Fatalf("want ok=false with message, got %+v body=%s", out.Data, w.Body.String())
	}
}

func TestStorageConfigTest_MissingBucket(t *testing.T) {
	svc := &attachment.Service{Resolve: func(context.Context) (attachment.MinIOConfig, error) {
		return attachment.MinIOConfig{Endpoint: "127.0.0.1:1"}, nil
	}}
	r := newStorageTestRouter(svc)
	mgr := auth.NewManager("test-secret", time.Hour)
	token, err := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	if err != nil {
		t.Fatal(err)
	}
	w := postJSONAuth(t, r, "/api/admin/v1/storage-config/test", `{}`, token)
	var out struct {
		Data struct {
			Ok      bool   `json:"ok"`
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, w.Body.String())
	}
	if out.Data.Message != "bucket 未配置" {
		t.Fatalf("message=%q, want bucket 未配置", out.Data.Message)
	}
}
