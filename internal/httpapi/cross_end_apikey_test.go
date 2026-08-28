package httpapi_test

// 跨端权限验收集(API key 主体篇):X-API-Key 三端主体准入矩阵。
// 边界契约(httpapi.go 头注):admin=account(全量 RBAC)/worker/customer(受限);
// user=customer 仅;worker=worker 仅。非本端主体一律拒绝。

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
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/httpapi"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

type crossEndAPIKey struct {
	apikey.Service
	subjects map[string]*apikey.Subject // sha256hex → subject
}

func (f *crossEndAPIKey) Lookup(_ context.Context, hash string) (*apikey.Subject, error) {
	if s, ok := f.subjects[hash]; ok {
		return s, nil
	}
	return nil, apikey.ErrNotFound
}
func (f *crossEndAPIKey) Touch(context.Context, string) {}

type crossEndWorker struct {
	worker.WorkerService
}

func (crossEndWorker) GetWorker(_ context.Context, id int64) (*worker.Worker, error) {
	return &worker.Worker{ID: id, Name: "张师傅"}, nil
}

type crossEndCustomer struct {
	customer.CustomerService
}

func (crossEndCustomer) Get(_ context.Context, id int64) (*customer.Customer, error) {
	return &customer.Customer{ID: id, Name: "客户甲"}, nil
}

// newAPIKeyServer 带 API key 桩的三端注册。
func newAPIKeyServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	keyFor := func(subjType string, ref int64) (plain, hash string) {
		plain = "k-" + subjType
		b := sha256.Sum256([]byte(plain))
		return plain, hex.EncodeToString(b[:])
	}
	subjects := map[string]*apikey.Subject{}
	for _, tc := range []struct {
		typ string
		ref int64
	}{
		{apikey.SubjectAccount, 1}, {apikey.SubjectWorker, 9}, {apikey.SubjectCustomer, 7},
	} {
		_, h := keyFor(tc.typ, tc.ref)
		subjects[h] = &apikey.Subject{Type: tc.typ, Ref: tc.ref}
	}
	httpapi.RegisterRoutes(r, &app.Application{
		User:     permUser{}, // HasPermission 恒 false:受限主体不得过 RBAC
		Points:   crossEndLoy{},
		Worker:   crossEndWorker{},
		Customer: crossEndCustomer{},
		APIKey:   &crossEndAPIKey{subjects: subjects},
	}, auth.NewManager("test-secret", time.Hour))
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)
	return ts
}

// permUser HasPermission 恒 false:非 account 主体(无 RBAC)在权限门前必 403。
type permUser struct {
	user.Service
}

func (permUser) GetProfile(_ context.Context, accountID int64) (*user.Profile, error) {
	return &user.Profile{AccountID: accountID, Username: "ops", RoleCode: "ADMIN"}, nil
}

func (permUser) AccountActive(context.Context, int64) (bool, error) { return true, nil }
func (permUser) HasPermission(context.Context, int64, string) (bool, error) {
	return false, nil
}

// verdict 验收集判定口径:放行/拒绝(不区分端间 HTTP 401 与业务码 40100 的风格差)。
type verdict bool

const (
	denied  verdict = false
	allowed verdict = true
)

// apiKeyDo 鉴权结论:HTTP 401/403 或 body.code≠0 视为拒绝;HTTP 200 且 code=0 视为放行。
func apiKeyDo(ts *httptest.Server, method, path, key string) verdict {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("X-API-Key", key)
	w := httptest.NewRecorder()
	ts.Config.Handler.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
		return denied
	}
	var body struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code == http.StatusOK && body.Code == 0
}

// TestCrossEnd_APIKeySubjectMatrix 主体×端准入矩阵。
func TestCrossEnd_APIKeySubjectMatrix(t *testing.T) {
	ts := newAPIKeyServer(t)
	cases := []struct {
		name string
		key  string
		path string
		want verdict
	}{
		// worker 主体:user 端拒(customer-only 门)、admin RBAC 端 403(无 RBAC)、worker 端放行。
		{"worker key on user end", "k-worker", "/api/user/v1/points", denied},
		{"worker key on admin rbac end", "k-worker", "/api/admin/v1/coupon-templates", denied},
		{"worker key on worker end", "k-worker", "/api/worker/v1/help/faq", allowed},
		// customer 主体:admin RBAC 端 403、user 端放行、worker 端拒(resolver 只收 worker)。
		{"customer key on admin rbac end", "k-customer", "/api/admin/v1/coupon-templates", denied},
		{"customer key on user end", "k-customer", "/api/user/v1/points", allowed},
		{"customer key on worker end", "k-customer", "/api/worker/v1/help/faq", denied},
		// account 主体:worker/user 端非本端主体拒;admin RBAC 端过认证,权限由 RBAC 定(此处 stub 恒拒 403)。
		{"account key on worker end", "k-account", "/api/worker/v1/help/faq", denied},
		{"account key on user end", "k-account", "/api/user/v1/points", denied},
		// 无效 key 三端全 401。
		{"bogus key", "k-bogus", "/api/admin/v1/auth/me", denied},
	}
	for _, tc := range cases {
		if got := apiKeyDo(ts, http.MethodGet, tc.path, tc.key); got != tc.want {
			t.Errorf("%s: got allowed=%v want %v", tc.name, got, tc.want)
		}
	}
}
