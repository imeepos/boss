package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/license"
)

// fakeLicenseService 可测的授权服务:AlwaysOK 时 Check 恒成功,否则恒失败。
type fakeLicenseService struct {
	checkErr error
}

func (f *fakeLicenseService) Check(_ context.Context) (license.Status, error) {
	if f.checkErr != nil {
		return license.Status{}, f.checkErr
	}
	return license.Status{Activated: true}, nil
}

func TestLicenseGateNilService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/admin/v1/x", LicenseGate(nil), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("nil service should pass through, got %d", w.Code)
	}
}

// TestLicenseGateNilPointerInInterface nil 指针装箱为接口后不得触发门禁
// (回归:app.License 为 nil *license.Service 时曾误拦截并 panic)。
func TestLicenseGateNilPointerInInterface(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var svc *fakeLicenseService // nil 指针
	r := gin.New()
	r.GET("/api/admin/v1/x", LicenseGate(svc), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("nil pointer in interface should pass through, got %d", w.Code)
	}
}

func TestLicenseGateBlocks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/admin/v1/x",
		LicenseGate(&fakeLicenseService{checkErr: errors.New("no license")}),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", w.Code)
	}
	if w.Body.String() == "" || !containsStr(w.Body.String(), "LICENSE_REQUIRED") {
		t.Fatalf("want LICENSE_REQUIRED body, got %s", w.Body.String())
	}
}

func TestLicenseGateExempt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/x")
	g.Use(LicenseGate(&fakeLicenseService{checkErr: errors.New("no license")}, "/x/license/"))
	g.GET("/biz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	g.GET("/license/status", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x/biz", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("biz path should be blocked, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/x/license/status", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("exempt path should pass, got %d", w.Code)
	}
}

func TestLicenseGateAllows(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/admin/v1/x",
		LicenseGate(&fakeLicenseService{}),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/x", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("valid license should pass, got %d", w.Code)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}