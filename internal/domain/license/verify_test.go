package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// signTokenBytes 用私钥构造合法令牌(底层实现,不依赖 t)。
func signTokenBytes(priv ed25519.PrivateKey, claims Claims) []byte {
	payload, _ := json.Marshal(claims)
	sig := ed25519.Sign(priv, payload)
	tok := Token{
		Payload:   base64.StdEncoding.EncodeToString(payload),
		KeyID:     "test-key",
		Signature: hex.EncodeToString(sig),
		Algorithm: "ed25519",
	}
	data, _ := json.Marshal(tok)
	return data
}

// signToken 测试辅助:模拟 release-platform 签发。
func signToken(t *testing.T, priv ed25519.PrivateKey, claims Claims) []byte {
	t.Helper()
	return signTokenBytes(priv, claims)
}

// fixtureClaims 以当前时间为中心构造 claims(NotBefore=1h 前,ExpiresAt=30d 后),
// 保证 Service.Check 用 time.Now() 时永远位于有效窗内。
func fixtureClaims() Claims {
	now := time.Now().UTC()
	return Claims{
		LicenseID:       "ac-test-1",
		ProductID:       "boss-server",
		DeviceID:        "boss-license-dev-01",
		FingerprintHash: "fp-boss-01",
		LicenseType:     "duration",
		Status:          "consumed",
		ExpiresAt:       ptrTime(now.Add(30 * 24 * time.Hour)),
		IssuedAt:        now.Add(-time.Hour),
		NotBefore:       now.Add(-time.Hour),
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

func newVerifier(t *testing.T, priv ed25519.PrivateKey) *Verifier {
	t.Helper()
	pub := priv.Public().(ed25519.PublicKey)
	v, err := NewVerifier(hex.EncodeToString(pub))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestVerifyHappyPath(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	v := newVerifier(t, priv)
	data := signToken(t, priv, fixtureClaims())
	got, err := v.Verify(data, VerifyOptions{Now: time.Now().UTC()})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.LicenseID != "ac-test-1" || got.Status != "consumed" {
		t.Fatalf("unexpected claims: %+v", got)
	}
}

func TestVerifyBadSignature(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	_, otherPriv, _ := ed25519.GenerateKey(rand.Reader)
	v := newVerifier(t, otherPriv) // 错误公钥
	data := signToken(t, priv, fixtureClaims())
	_, err := v.Verify(data, VerifyOptions{Now: time.Now().UTC()})
	if !errors.Is(err, ErrBadSignature) {
		t.Fatalf("want ErrBadSignature, got %v", err)
	}
}

func TestVerifyWrongDevice(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	v := newVerifier(t, priv)
	data := signToken(t, priv, fixtureClaims())
	_, err := v.Verify(data, VerifyOptions{
		Now:                 time.Date(2026, 8, 27, 13, 0, 0, 0, time.UTC),
		ExpectedDeviceID:    "other-device",
		ExpectedFingerprint: "fp-boss-01",
	})
	if !errors.Is(err, ErrWrongDevice) {
		t.Fatalf("want ErrWrongDevice, got %v", err)
	}
}

func TestVerifyExpired(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	v := newVerifier(t, priv)
	c := fixtureClaims()
	c.GraceEndsAt = nil
	data := signToken(t, priv, c)
	_, err := v.Verify(data, VerifyOptions{
		Now: c.ExpiresAt.Add(24 * time.Hour),
	})
	if !errors.Is(err, ErrExpired) {
		t.Fatalf("want ErrExpired, got %v", err)
	}
}

func TestVerifyGraceWindow(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	v := newVerifier(t, priv)
	c := fixtureClaims()
	grace := c.ExpiresAt.Add(7 * 24 * time.Hour)
	c.GraceEndsAt = &grace
	data := signToken(t, priv, c)
	_, err := v.Verify(data, VerifyOptions{
		Now: c.ExpiresAt.Add(24 * time.Hour), // 已过期但在宽限内
	})
	if err != nil {
		t.Fatalf("grace window should pass, got %v", err)
	}
}

func TestVerifyRevoked(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	v := newVerifier(t, priv)
	c := fixtureClaims()
	c.Status = "revoked"
	data := signToken(t, priv, c)
	_, err := v.Verify(data, VerifyOptions{Now: time.Now().UTC()})
	if !errors.Is(err, ErrRevoked) {
		t.Fatalf("want ErrRevoked, got %v", err)
	}
}

func TestVerifyNotYetValid(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	v := newVerifier(t, priv)
	c := fixtureClaims()
	c.NotBefore = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	data := signToken(t, priv, c)
	_, err := v.Verify(data, VerifyOptions{Now: time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)})
	if !errors.Is(err, ErrNotYetValid) {
		t.Fatalf("want ErrNotYetValid, got %v", err)
	}
}

func TestVerifyMalformedToken(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	v := newVerifier(t, priv)
	_, err := v.Verify([]byte(`{"payload":"!!!not-base64"}`), VerifyOptions{Now: time.Now()})
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("want ErrInvalidToken, got %v", err)
	}
}

func TestNewVerifierBadKey(t *testing.T) {
	if _, err := NewVerifier("abcd"); err == nil {
		t.Fatal("want error for bad pubkey hex")
	}
}
