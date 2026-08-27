package adminapi

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/license"
)

// newLicenseApp 组装含真实 License 服务的 Application(仅授权相关字段)。
func newLicenseApp(t *testing.T) (*app.Application, *license.Service) {
	t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	pub := priv.Public().(ed25519.PublicKey)
	svc := &license.Service{
		Verifier: mustVerifier(t, pub),
		Store:    &license.Store{Path: filepath.Join(t.TempDir(), "license.json")},
		Cfg: license.Config{
			PublicKeyHex: hex.EncodeToString(pub),
			DeviceID:     "boss-license-dev-01",
			Fingerprint:  "fp-boss-01",
		},
		API: fakeLicActivator{priv: priv},
	}
	return &app.Application{License: svc}, svc
}

// newLicenseAppDisabled License 为 nil(未启用门禁)。
func newLicenseAppDisabled() *app.Application {
	return &app.Application{License: nil}
}

func mustVerifier(t *testing.T, pub ed25519.PublicKey) *license.Verifier {
	t.Helper()
	v, err := license.NewVerifier(hex.EncodeToString(pub))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestLicenseStatusNotActivated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a, _ := newLicenseApp(t)
	r := gin.New()
	registerLicenseRoutes(r.Group("/api/admin/v1"), a)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/license/status", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if !containsStr(w.Body.String(), `"activated":false`) {
		t.Fatalf("want not activated, got %s", w.Body.String())
	}
}

func TestLicenseActivateFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a, _ := newLicenseApp(t)
	r := gin.New()
	registerLicenseRoutes(r.Group("/api/admin/v1"), a)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"activationCode":"TEST-CODE-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/license/activate", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", w.Code, w.Body.String())
	}
	// 激活后 status 应为已激活(data.activated=true)。
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/admin/v1/license/status", nil)
	r.ServeHTTP(w, req)
	if !containsStr(w.Body.String(), `"activated":true`) {
		t.Fatalf("want activated after activate, got %s", w.Body.String())
	}
}

func TestLicenseActivateBadBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a, _ := newLicenseApp(t)
	r := gin.New()
	registerLicenseRoutes(r.Group("/api/admin/v1"), a)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/license/activate", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	// 恒 200 envelope,业务错误在 code/msg 里。
	if w.Code != http.StatusOK {
		t.Fatalf("want 200 envelope, got %d", w.Code)
	}
	if !containsStr(w.Body.String(), "activationCode required") {
		t.Fatalf("want activationCode required, got %s", w.Body.String())
	}
}

func TestLicenseStatusDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a := newLicenseAppDisabled()
	r := gin.New()
	registerLicenseRoutes(r.Group("/api/admin/v1"), a)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/license/status", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if !containsStr(w.Body.String(), `"enabled":false`) {
		t.Fatalf("want disabled status, got %s", w.Body.String())
	}
}

// fakeLicActivator 测试激活器:签发与 priv 匹配的令牌。
type fakeLicActivator struct{ priv ed25519.PrivateKey }

func (f fakeLicActivator) Exchange(_ context.Context, code, productID, deviceID, fingerprint string) (*license.Claims, []byte, error) {
	now := time.Now().UTC()
	c := license.Claims{
		LicenseID: "ac-test-1", ProductID: productID,
		DeviceID: deviceID, FingerprintHash: fingerprint,
		LicenseType: "duration", Status: "consumed",
		ExpiresAt: ptrLicTime(now.Add(30 * 24 * time.Hour)),
		IssuedAt:  now.Add(-time.Hour),
		NotBefore: now.Add(-time.Hour),
	}
	return &c, signLicenseBytes(f.priv, c), nil
}

func ptrLicTime(t time.Time) *time.Time { return &t }

// signLicenseBytes 构造真实签发令牌:base64(payload) + hex(ed25519 签名)。
func signLicenseBytes(priv ed25519.PrivateKey, c license.Claims) []byte {
	payload, _ := json.Marshal(c)
	sig := ed25519.Sign(priv, payload)
	tok := license.Token{
		Payload:   base64.StdEncoding.EncodeToString(payload),
		KeyID:     "test-key",
		Signature: hex.EncodeToString(sig),
		Algorithm: "ed25519",
	}
	data, _ := json.Marshal(tok)
	return data
}

// containsStr 简单子串判断。
func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && indexOfStr(s, sub) >= 0
}

func indexOfStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}