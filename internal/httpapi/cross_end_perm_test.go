package httpapi_test

// 跨端权限验收集(2028 Q2 交付):三端身份边界端到端断言。
// 单一 gin engine 注册 admin/user/worker 三端,验证:
//  1. JWT aud 单向精确匹配(跨端 token 一律 401);
//  2. workerJWT(issuer=boss-worker)不被 admin/user 端接受,反之亦然;
//  3. 无 token / 过期 token / 伪造签名 token 全部 401;
//  4. 正向对照:正确 token 在本端可达(排除 401 来自路由缺失)。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/loy"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/httpapi"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/config"
)

// crossEndUser 覆盖 admin 正向对照所需的最小 user.Service 方法。
type crossEndUser struct {
	user.Service
}

func (crossEndUser) GetProfile(_ context.Context, accountID int64) (*user.Profile, error) {
	return &user.Profile{AccountID: accountID, Username: "ops", RoleCode: "ADMIN"}, nil
}

// crossEndLoy 覆盖 user 端 GET /points 所需方法,其余经嵌入零值不触达。
type crossEndLoy struct {
	loy.Service
}

func (crossEndLoy) Balance(context.Context, int64) (int64, error) { return 100, nil }
func (crossEndLoy) Entries(context.Context, int64) ([]loy.Entry, error) {
	return []loy.Entry{}, nil
}

type workerClaimsForTest struct {
	WorkerID   int64  `json:"wid"`
	WorkerName string `json:"wname"`
	jwt.RegisteredClaims
}

// forgeWorkerToken 与 worker 端 signWorkerToken 同构(issuer=boss-worker,subject=worker)。
func forgeWorkerToken(secret string) string {
	now := time.Now()
	c := workerClaimsForTest{
		WorkerID: 9, WorkerName: "张师傅",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			Subject:   "worker", Issuer: "boss-worker",
		},
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(secret))
	return tok
}

// newCrossEndServer 三端全量注册 + 最小桩;返回 server 与三种 token。
func newCrossEndServer(t *testing.T) (ts *httptest.Server, adminTok, userTok, workerTok string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	secret := config.Load().JWT.Secret
	mgr := auth.NewManager(secret, time.Hour)
	r := gin.New()
	httpapi.RegisterRoutes(r, &app.Application{
		User:   crossEndUser{},
		Points: crossEndLoy{},
	}, mgr)

	adminTok, _ = mgr.Sign(auth.AudAdmin, 1, "ops", "ADMIN")
	// 与 userapi.signCustomerToken 同构:accountID=0 + role=customer + username=cust/<id>/<phone>。
	userTok, _ = mgr.Sign(auth.AudUser, 0, "cust/7/13800001234", "customer")
	workerTok = forgeWorkerToken(secret)
	ts = httptest.NewServer(r)
	t.Cleanup(ts.Close)
	return ts, adminTok, userTok, workerTok
}

func crossEndDo(ts *httptest.Server, method, path, token string) int {
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	// httptest.NewRequest 的 req 已带 path,直接走 server handler。
	ts.Config.Handler.ServeHTTP(w, req)
	return w.Code
}

// TestCrossEnd_TokenAudienceIsolation 跨端 token 冒用全部 401(验收集核心)。
func TestCrossEnd_TokenAudienceIsolation(t *testing.T) {
	ts, adminTok, userTok, workerTok := newCrossEndServer(t)
	cases := []struct {
		name  string
		token string
		path  string
	}{
		{"admin token 打 user 端", adminTok, "/api/user/v1/points"},
		{"admin token 打 worker 端", adminTok, "/api/worker/v1/help/faq"},
		{"user token 打 admin 端", userTok, "/api/admin/v1/auth/me"},
		{"user token 打 worker 端", userTok, "/api/worker/v1/help/faq"},
		{"worker token 打 admin 端", workerTok, "/api/admin/v1/auth/me"},
		{"worker token 打 user 端", workerTok, "/api/user/v1/points"},
	}
	for _, tc := range cases {
		if code := crossEndDo(ts, http.MethodGet, tc.path, tc.token); code != http.StatusUnauthorized {
			t.Errorf("%s: got %d want 401", tc.name, code)
		}
	}
}

// TestCrossEnd_NoTokenRejected 无 token 三端全 401。
func TestCrossEnd_NoTokenRejected(t *testing.T) {
	ts, _, _, _ := newCrossEndServer(t)
	for _, p := range []string{
		"/api/admin/v1/auth/me", "/api/user/v1/points", "/api/worker/v1/help/faq",
	} {
		if code := crossEndDo(ts, http.MethodGet, p, ""); code != http.StatusUnauthorized {
			t.Errorf("%s: got %d want 401", p, code)
		}
	}
}

// TestCrossEnd_MalformedTokens 垃圾/伪造签名/过期 token 全 401。
func TestCrossEnd_MalformedTokens(t *testing.T) {
	ts, _, _, _ := newCrossEndServer(t)
	garbage := "Bearer not-a-jwt"
	wrongSign := func() string {
		m := auth.NewManager("other-secret", time.Hour)
		tok, _ := m.Sign(auth.AudUser, 0, "cust/7/x", "customer")
		return tok
	}()
	expired := func() string {
		m := auth.NewManager(config.Load().JWT.Secret, -time.Minute)
		tok, _ := m.Sign(auth.AudUser, 0, "cust/7/x", "customer")
		return tok
	}()
	for name, tok := range map[string]string{
		"garbage": garbage, "wrong signature": wrongSign, "expired": expired,
	} {
		for _, p := range []string{"/api/admin/v1/auth/me", "/api/user/v1/points"} {
			req := httptest.NewRequest(http.MethodGet, p, nil)
			if name == "garbage" {
				req.Header.Set("Authorization", tok)
			} else {
				req.Header.Set("Authorization", "Bearer "+tok)
			}
			w := httptest.NewRecorder()
			ts.Config.Handler.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("%s on %s: got %d want 401", name, p, w.Code)
			}
		}
	}
}

// TestCrossEnd_PositiveControls 正确 token 在本端可达(200),证明 401 来自鉴权而非路由缺失。
func TestCrossEnd_PositiveControls(t *testing.T) {
	ts, adminTok, userTok, workerTok := newCrossEndServer(t)
	cases := []struct {
		name         string
		token, path  string
	}{
		{"admin 本端", adminTok, "/api/admin/v1/auth/me"},
		{"user 本端", userTok, "/api/user/v1/points"},
		{"worker 本端", workerTok, "/api/worker/v1/help/faq"},
	}
	for _, tc := range cases {
		if code := crossEndDo(ts, http.MethodGet, tc.path, tc.token); code != http.StatusOK {
			t.Errorf("%s (%s): got %d want 200", tc.name, tc.path, code)
		}
	}
}
