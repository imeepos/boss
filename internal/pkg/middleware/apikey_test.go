package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"time"
)

// fakeKeyService 模拟 apikey.Service(主体模型)。
// touchN 用原子计数:Touch 由中间件后台 goroutine 触发,与测试主协程并发。
type fakeKeyService struct {
	lookup map[string]*apikey.Subject
	touchN atomic.Int64
}

func (f *fakeKeyService) Create(ctx context.Context, st string, ref, by int64, n, tpl string) (*apikey.CreateResult, error) {
	return nil, nil
}
func (f *fakeKeyService) List(ctx context.Context) ([]apikey.APIKey, error) { return nil, nil }
func (f *fakeKeyService) ListTemplates(ctx context.Context) ([]apikey.PermissionTemplate, error) {
	return nil, nil
}
func (f *fakeKeyService) Revoke(ctx context.Context, id int64) error { return nil }
func (f *fakeKeyService) Lookup(ctx context.Context, keyHash string) (*apikey.Subject, error) {
	if s, ok := f.lookup[keyHash]; ok {
		return s, nil
	}
	return nil, apikey.ErrNotFound
}
func (f *fakeKeyService) Touch(ctx context.Context, keyHash string) { f.touchN.Add(1) }

// fakeResolver 模拟 app 层 SubjectResolver。
func fakeResolver(ctx context.Context, subjType string, ref int64) (string, string, error) {
	switch subjType {
	case "account":
		return "alice", "ops", nil
	case "worker":
		return "张师傅", "worker", nil
	case "customer":
		return "王先生", "customer", nil
	}
	return "", "", errors.New("unknown")
}

// newAPIKeyRouter 构造测试路由:/whoami 输出 claims + subject。
func newAPIKeyRouter(keys apikey.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	authed := r.Group("/api/v1")
	authed.Use(APIKeyAuth(keys, fakeResolver), Authn(nil, auth.AudAdmin))
	authed.GET("/whoami", func(c *gin.Context) {
		out := gin.H{}
		if v, ok := c.Get(CtxClaims); ok {
			if cl, ok := v.(*auth.Claims); ok {
				out["aid"] = cl.AccountID
				out["usr"] = cl.Username
				out["role"] = cl.RoleCode
			}
		}
		if s := SubjectFrom(c); s != nil {
			out["subjType"] = s.Type
			out["subjRef"] = s.Ref
			out["subjName"] = s.Name
		}
		c.JSON(http.StatusOK, out)
	})
	return r
}

func TestAPIKeyAccountSubject(t *testing.T) {
	keys := &fakeKeyService{lookup: map[string]*apikey.Subject{
		sha256Hash("boss_admin_key"): {Type: "account", Ref: 7},
	}}
	r := newAPIKeyRouter(keys)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/whoami", nil)
	req.Header.Set("X-API-Key", "boss_admin_key")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	// account 主体:AccountID=ref,RBAC 完整生效
	for _, want := range []string{`"aid":7`, `"usr":"alice"`, `"role":"ops"`, `"subjType":"account"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("body=%s 缺少 %s", body, want)
		}
	}
	for i := 0; i < 20 && keys.touchN.Load() != 1; i++ {
		time.Sleep(time.Millisecond)
	}
	if keys.touchN.Load() != 1 {
		t.Fatalf("touchN=%d want 1", keys.touchN.Load())
	}
}

func TestAPIKeyWorkerSubject(t *testing.T) {
	keys := &fakeKeyService{lookup: map[string]*apikey.Subject{
		sha256Hash("boss_worker_key"): {Type: "worker", Ref: 5},
	}}
	r := newAPIKeyRouter(keys)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/whoami", nil)
	req.Header.Set("X-API-Key", "boss_worker_key")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	// worker 主体:AccountID=0(RBAC 恒拒),Subject 携带真实师傅身份
	for _, want := range []string{`"aid":0`, `"role":"worker"`, `"subjType":"worker"`, `"subjRef":5`, `"subjName":"张师傅"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("body=%s 缺少 %s", body, want)
		}
	}
}

func TestAPIKeyInvalid(t *testing.T) {
	keys := &fakeKeyService{lookup: map[string]*apikey.Subject{}}
	r := newAPIKeyRouter(keys)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/whoami", nil)
	req.Header.Set("X-API-Key", "boss_wrong")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d want 401", w.Code)
	}
}

func TestAPIKeyMissingFallsThroughToJWT(t *testing.T) {
	keys := &fakeKeyService{lookup: map[string]*apikey.Subject{}}
	r := newAPIKeyRouter(keys)

	// 无 API key → 回退 JWT,Authn(nil, auth.AudAdmin) 无 Bearer → 401
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/whoami", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d want 401", w.Code)
	}
}

func TestAPIKeySubjectNotFound(t *testing.T) {
	// 密钥有效但主体已被删除:resolver 报错 → 401
	keys := &fakeKeyService{lookup: map[string]*apikey.Subject{
		sha256Hash("boss_orphan"): {Type: "worker", Ref: 999},
	}}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", APIKeyAuth(keys, func(ctx context.Context, s string, i int64) (string, string, error) {
		return "", "", errors.New("gone")
	}), func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-API-Key", "boss_orphan")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d want 401 (subject gone)", w.Code)
	}
}

// TestAuthn_AudMismatch 跨端 token(admin aud 打 user 端)必须 401,aud 防线单向精确匹配。
func TestAuthn_AudMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := auth.NewManager("s", time.Hour)
	r := gin.New()
	r.GET("/x", Authn(m, auth.AudUser), func(c *gin.Context) { c.Status(200) })

	tok, _ := m.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	if w.Code != 401 {
		t.Fatalf("no-token code=%d", w.Code)
	}
	w = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("admin-aud on user end code=%d", w.Code)
	}

	utok, _ := m.Sign(auth.AudUser, 0, "cust/7/138", "customer")
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer "+utok)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("user-aud on user end code=%d", w.Code)
	}
}
