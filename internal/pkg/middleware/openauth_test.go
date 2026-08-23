// OpenAuth + OpenRateLimit 中间件链测试(fake Service,不起 DB)。
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/openplat"
)

// fakeOpenPlat 最小 openplat.Service 桩:单应用,计数即返回值。
type fakeOpenPlat struct {
	app   *openplat.AuthContext
	appID string
	count atomic.Int64
}

func (f *fakeOpenPlat) CreateApp(context.Context, string, int, int, bool, int64) (*openplat.AppCreateResult, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeOpenPlat) ListApps(context.Context) ([]openplat.App, error) { return nil, nil }
func (f *fakeOpenPlat) SetAppStatus(context.Context, int64, int16) error { return nil }
func (f *fakeOpenPlat) ListSubscriptions(context.Context, int64) ([]openplat.Subscription, error) {
	return nil, nil
}
func (f *fakeOpenPlat) CreateSubscription(context.Context, int64, string, string) (*openplat.Subscription, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeOpenPlat) DeleteSubscription(context.Context, int64) error { return nil }
func (f *fakeOpenPlat) ListDeliveries(context.Context, int64) ([]openplat.Delivery, error) {
	return nil, nil
}
func (f *fakeOpenPlat) Requeue(context.Context, int64) error { return nil }

func (f *fakeOpenPlat) LookupActive(_ context.Context, appID string) (*openplat.AuthContext, error) {
	if appID != f.appID {
		return nil, openplat.ErrNotFound
	}
	return f.app, nil
}

func (f *fakeOpenPlat) TouchUsage(_ context.Context, id int64, _ time.Time) (int64, error) {
	return f.count.Add(1), nil
}

// newOpenRouter 装配 /api/open/v1 测试路由(ping 端点)。
func newOpenRouter(svc openplat.Service, limiter *OpenRateLimiter) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/open/v1")
	v1.Use(OpenAuth(svc), limiter.OpenRateLimit())
	v1.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"appRowId": OpenAppIDFrom(c)})
	})
	return r
}

// signedRequest 构造带 HMAC 签名头的请求。
func signedRequest(t *testing.T, secret, appID, path string) *http.Request {
	t.Helper()
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	body := []byte("")
	req := httptest.NewRequest(http.MethodGet, path, nil)
	sig := openplat.Sign(secret, openplat.CanonicalString(appID, http.MethodGet, path, ts, "n1", body))
	req.Header.Set("X-BOSS-AppId", appID)
	req.Header.Set("X-BOSS-Timestamp", ts)
	req.Header.Set("X-BOSS-Nonce", "n1")
	req.Header.Set("X-BOSS-Signature", sig)
	return req
}

func TestOpenAuthPassesAndInjectsApp(t *testing.T) {
	svc := &fakeOpenPlat{appID: "op_a", app: &openplat.AuthContext{AppRowID: 9, Secret: "s", RateLimitRPM: 60, DailyQuota: 100}}
	r := newOpenRouter(svc, NewOpenRateLimiter())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, signedRequest(t, "s", "op_a", "/api/open/v1/ping"))
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var got struct {
		AppRowID int64 `json:"appRowId"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.AppRowID != 9 {
		t.Fatalf("app row not injected: %+v", got)
	}
}

func TestOpenAuthRejectsBadSignature(t *testing.T) {
	svc := &fakeOpenPlat{appID: "op_a", app: &openplat.AuthContext{AppRowID: 9, Secret: "s"}}
	r := newOpenRouter(svc, NewOpenRateLimiter())
	req := signedRequest(t, "WRONG", "op_a", "/api/open/v1/ping")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestOpenAuthRejectsUnknownApp(t *testing.T) {
	svc := &fakeOpenPlat{appID: "op_a", app: &openplat.AuthContext{Secret: "s"}}
	r := newOpenRouter(svc, NewOpenRateLimiter())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, signedRequest(t, "s", "op_ghost", "/api/open/v1/ping"))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestOpenAuthRejectsMissingHeaders(t *testing.T) {
	svc := &fakeOpenPlat{appID: "op_a", app: &openplat.AuthContext{Secret: "s"}}
	r := newOpenRouter(svc, NewOpenRateLimiter())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/open/v1/ping", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestOpenRateLimitReturns429(t *testing.T) {
	svc := &fakeOpenPlat{appID: "op_a", app: &openplat.AuthContext{AppRowID: 1, Secret: "s", RateLimitRPM: 2, DailyQuota: 1000}}
	r := newOpenRouter(svc, NewOpenRateLimiter())

	saw429 := false
	for i := 0; i < 4; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, signedRequest(t, "s", "op_a", "/api/open/v1/ping"))
		if w.Code == http.StatusTooManyRequests {
			saw429 = true
		}
	}
	if !saw429 {
		t.Fatal("rpm=2 burst should trigger 429 within 4 calls")
	}
}

func TestOpenQuotaExceededReturns429(t *testing.T) {
	svc := &fakeOpenPlat{appID: "op_a", app: &openplat.AuthContext{AppRowID: 1, Secret: "s", RateLimitRPM: 1000, DailyQuota: 2}}
	r := newOpenRouter(svc, NewOpenRateLimiter())

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, signedRequest(t, "s", "op_a", "/api/open/v1/ping"))
		if i == 2 && w.Code != http.StatusTooManyRequests {
			t.Fatalf("call %d: want 429 quota, got %d", i, w.Code)
		}
	}
	if svc.count.Load() != 3 {
		t.Fatalf("usage should be counted 3 times, got %d", svc.count.Load())
	}
}
