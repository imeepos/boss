package secretbox

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestSealOpenRoundTrip(t *testing.T) {
	t.Setenv("BOSS_AUTH_SECRET_KEY", "test-key-a")
	ct, err := Seal("hunter2")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if !strings.HasPrefix(ct, "enc:v1:") {
		t.Fatalf("Seal 缺少前缀: %q", ct)
	}
	pt, err := Open(ct)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if pt != "hunter2" {
		t.Fatalf("Open=%q, want hunter2", pt)
	}
}

func TestSealEmpty(t *testing.T) {
	got, err := Seal("")
	if err != nil || got != "" {
		t.Fatalf("Seal(\"\")=(%q,%v), want (\"\",nil)", got, err)
	}
}

func TestSealRandomized(t *testing.T) {
	t.Setenv("BOSS_AUTH_SECRET_KEY", "test-key-a")
	a, _ := Seal("same-input")
	b, _ := Seal("same-input")
	if a == b {
		t.Fatalf("两次 Seal 输出相同, nonce 未随机化: %q", a)
	}
}

func TestOpenPlaintextCompat(t *testing.T) {
	for _, s := range []string{"", "plain-secret", "enc:v0:legacy"} {
		got, err := Open(s)
		if err != nil || got != s {
			t.Fatalf("Open(%q)=(%q,%v), want 原样返回", s, got, err)
		}
	}
}

func TestOpenBadCiphertext(t *testing.T) {
	t.Setenv("BOSS_AUTH_SECRET_KEY", "test-key-a")
	tests := []struct {
		name string
		s    string
	}{
		{name: "非 base64", s: "enc:v1:!!!not-base64!!!"},
		{name: "密文短于 nonce", s: "enc:v1:" + encodeShort()},
		{name: "认证失败", s: "enc:v1:" + tamperedCiphertext(t)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Open(tt.s); err == nil {
				t.Fatalf("Open(%q) err=nil, want error", tt.s)
			}
		})
	}
}

func TestOpenWrongKey(t *testing.T) {
	t.Setenv("BOSS_AUTH_SECRET_KEY", "test-key-a")
	ct, err := Seal("secret")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	t.Setenv("BOSS_AUTH_SECRET_KEY", "test-key-b")
	if _, err := Open(ct); err == nil {
		t.Fatalf("换密钥后 Open 应失败")
	}
}

func TestKeyEnvFallback(t *testing.T) {
	t.Run("回退 BOSS_JWT_SECRET", func(t *testing.T) {
		os.Setenv("BOSS_AUTH_SECRET_KEY", "")
		t.Cleanup(func() { os.Unsetenv("BOSS_JWT_SECRET") })
		t.Setenv("BOSS_JWT_SECRET", "jwt-secret")
		if want := Key(); !bytes.Equal(Key(), want) {
			t.Fatalf("Key 不稳定")
		}
	})
	t.Run("默认密钥", func(t *testing.T) {
		os.Setenv("BOSS_AUTH_SECRET_KEY", "")
		os.Setenv("BOSS_JWT_SECRET", "")
		if len(Key()) != 32 {
			t.Fatalf("len(Key())=%d, want 32", len(Key()))
		}
	})
}

func TestSealNonceReaderError(t *testing.T) {
	t.Setenv("BOSS_AUTH_SECRET_KEY", "test-key-a")
	orig := rand.Reader
	rand.Reader = errReader{}
	t.Cleanup(func() { rand.Reader = orig })
	if _, err := Seal("x"); err == nil || !strings.Contains(err.Error(), "nonce") {
		t.Fatalf("Seal err=%v, want nonce error", err)
	}
}

func TestNewGCMError(t *testing.T) {
	orig := keyFn
	keyFn = func() []byte { return []byte("short") } // 非 16/24/32 字节,aes.NewCipher 必失败
	t.Cleanup(func() { keyFn = orig })
	if _, err := Seal("x"); err == nil {
		t.Fatalf("Seal err=nil, want newGCM error")
	}
	if _, err := Open("enc:v1:QUJD"); err == nil {
		t.Fatalf("Open err=nil, want newGCM error")
	}
	if _, err := newGCM(); err == nil {
		t.Fatalf("newGCM err=nil, want error")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func encodeShort() string {
	// 3 字节 < gcm.NonceSize(12)
	return base64.StdEncoding.EncodeToString([]byte("abc"))
}

func tamperedCiphertext(t *testing.T) string {
	t.Helper()
	ct, err := Seal("payload")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	raw := ct[len("enc:v1:"):]
	b, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	b[len(b)-1] ^= 0xff
	return base64.StdEncoding.EncodeToString(b)
}
