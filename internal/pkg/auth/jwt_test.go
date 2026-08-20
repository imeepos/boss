package auth

// Manager 单测:签发/校验正反路径(篡改、过期、跨密钥、非 HMAC 算法、非本系统 issuer)。

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestManagerSignVerify(t *testing.T) {
	m := NewManager("secret", time.Hour)

	tok, err := m.Sign(AudAdmin, 42, "alice", "ADMIN")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	c, err := m.Verify(tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if c.AccountID != 42 || c.Username != "alice" || c.RoleCode != "ADMIN" || c.Aud != AudAdmin {
		t.Fatalf("claims mismatch: %+v", c)
	}
	if c.Issuer != "boss" || c.KeyID != "" {
		t.Fatalf("issuer=%q kid=%q", c.Issuer, c.KeyID)
	}
}

func TestManagerVerifyInvalid(t *testing.T) {
	m := NewManager("secret", time.Hour)
	valid, err := m.Sign(AudUser, 1, "bob", "USER")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	// 过期 token(负 TTL 签出即失效)。
	expired, err := NewManager("secret", -time.Minute).Sign(AudUser, 1, "bob", "USER")
	if err != nil {
		t.Fatalf("Sign expired: %v", err)
	}

	// 非 HMAC 算法(none),keyfunc 应拒绝。
	noneTok, err := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
		Aud: AudAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    "boss",
		},
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none: %v", err)
	}

	// 同密钥但 issuer 非本系统。
	foreign, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		Aud: AudAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    "other-system",
		},
	}).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign foreign: %v", err)
	}

	cases := []struct {
		name  string
		token string
	}{
		{"垃圾字符串", "not-a-jwt"},
		{"篡改载荷", valid + "x"},
		{"错误密钥", mustSign(t, NewManager("other", time.Hour))},
		{"过期", expired},
		{"非HMAC算法", noneTok},
		{"非本系统issuer", foreign},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := m.Verify(tc.token); err != ErrInvalidToken {
				t.Fatalf("err=%v, want ErrInvalidToken", err)
			}
		})
	}
}

func mustSign(t *testing.T, m *Manager) string {
	t.Helper()
	tok, err := m.Sign(AudAdmin, 1, "u", "ADMIN")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	return tok
}

// 空 aud 也能签发/校验(端校验在上层),确保无 aud 声明时 RegisteredClaims.Audience 不干扰。
func TestSignEmptyAud(t *testing.T) {
	m := NewManager("s", time.Hour)
	tok, err := m.Sign("", 7, "u", "USER")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if strings.Count(tok, ".") != 2 {
		t.Fatalf("token %q not compact JWS", tok)
	}
	if _, err := m.Verify(tok); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}
