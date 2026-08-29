package adminapi

// /auth/me 主体真相测试:受限模板 API key 必须回模板授权码集(Authz 恒拒模板外权限),
// 普通 account key 回角色全集,worker/customer 主体回主体身份。bossmcp admin 目录过滤依赖此契约。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// meTemplateAPIKey 桩 apikey.Service:固定主体表 + 固定模板授权集。
type meTemplateAPIKey struct {
	apikey.Service
	subjects map[string]*apikey.Subject // sha256hex → subject
}

func (f *meTemplateAPIKey) Lookup(_ context.Context, hash string) (*apikey.Subject, error) {
	if s, ok := f.subjects[hash]; ok {
		return s, nil
	}
	return nil, apikey.ErrNotFound
}

func (f *meTemplateAPIKey) ListTemplates(context.Context) ([]apikey.PermissionTemplate, error) {
	return []apikey.PermissionTemplate{
		{Code: "partner-orders-read", Permissions: []string{"menu:partner-orders"}},
	}, nil
}

func (f *meTemplateAPIKey) Touch(context.Context, string) {}

// meWorker 桩 worker 服务:主体解析仅取姓名。
type meWorker struct {
	worker.WorkerService
}

func (meWorker) GetWorker(_ context.Context, id int64) (*worker.Worker, error) {
	return &worker.Worker{ID: id, Name: "王测试"}, nil
}

// newMeRouter 装配带 API key 桩的 admin 路由。
func newMeRouter(f *fakeUser, keys *meTemplateAPIKey) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{User: f, APIKey: keys, Worker: meWorker{}}, auth.NewManager("test-secret", time.Hour))
	return r
}

func getKey(t *testing.T, plain string) string {
	t.Helper()
	b := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(b[:])
}

// getMe GET /auth/me 并解出 data。
func getMe(t *testing.T, r *gin.Engine, apiKey string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/auth/me", nil)
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Code != 0 {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	return body.Data
}

// codesOf data.permissionCodes 转字符串集。
func codesOf(t *testing.T, data map[string]any) map[string]bool {
	t.Helper()
	raw, ok := data["permissionCodes"].([]any)
	if !ok {
		t.Fatalf("permissionCodes missing: %v", data)
	}
	out := map[string]bool{}
	for _, v := range raw {
		out[v.(string)] = true
	}
	return out
}

func TestAuthMeTemplateTruth(t *testing.T) {
	f := &fakeUser{}
	keys := &meTemplateAPIKey{subjects: map[string]*apikey.Subject{
		getKey(t, "tpl-key"):    {Type: apikey.SubjectAccount, Ref: 1, TemplateCode: "partner-orders-read"},
		getKey(t, "acct-key"):   {Type: apikey.SubjectAccount, Ref: 1},
		getKey(t, "worker-key"): {Type: apikey.SubjectWorker, Ref: 9},
	}}
	r := newMeRouter(f, keys)

	t.Run("模板key回模板授权码集", func(t *testing.T) {
		data := getMe(t, r, "tpl-key")
		codes := codesOf(t, data)
		if len(codes) != 1 || !codes["menu:partner-orders"] {
			t.Fatalf("permissionCodes=%v, want [menu:partner-orders]", codes)
		}
		if data["templateCode"] != "partner-orders-read" {
			t.Fatalf("templateCode=%v", data["templateCode"])
		}
	})

	t.Run("普通accountkey回角色全集且无templateCode", func(t *testing.T) {
		data := getMe(t, r, "acct-key")
		if _, has := data["templateCode"]; has {
			t.Fatalf("templateCode should be absent: %v", data["templateCode"])
		}
	})

	t.Run("worker主体回主体身份", func(t *testing.T) {
		data := getMe(t, r, "worker-key")
		if data["subjectType"] != "worker" {
			t.Fatalf("data=%v", data)
		}
	})
}
