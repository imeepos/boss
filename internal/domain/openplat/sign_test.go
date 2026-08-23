package openplat

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSignVerifyRoundTrip(t *testing.T) {
	const secret = "ops_deadbeef"
	body := []byte(`{"a":1}`)
	ts := time.Now().Unix()
	canonical := CanonicalString("op_x", "GET", "/api/open/v1/ping", itoa(ts), "nonce-1", body)
	sig := Sign(secret, canonical)

	r := httptest.NewRequest(http.MethodGet, "/api/open/v1/ping", nil)
	auth := &AuthContext{AppRowID: 1, Secret: secret}
	if err := VerifyRequest(auth, r, body, "op_x", itoa(ts), "nonce-1", sig); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestVerifyRejectsTamperedBody(t *testing.T) {
	const secret = "ops_deadbeef"
	ts := time.Now().Unix()
	canonical := CanonicalString("op_x", "GET", "/api/open/v1/ping", itoa(ts), "n", []byte("orig"))
	sig := Sign(secret, canonical)
	r := httptest.NewRequest(http.MethodGet, "/api/open/v1/ping", nil)
	auth := &AuthContext{Secret: secret}
	err := VerifyRequest(auth, r, []byte("tampered"), "op_x", itoa(ts), "n", sig)
	if err != ErrBadSignature {
		t.Fatalf("want ErrBadSignature, got %v", err)
	}
}

func TestVerifyRejectsStaleTimestamp(t *testing.T) {
	old := time.Now().Add(-10 * time.Minute).Unix()
	r := httptest.NewRequest(http.MethodGet, "/api/open/v1/ping", nil)
	auth := &AuthContext{Secret: "s"}
	err := VerifyRequest(auth, r, nil, "op_x", itoa(old), "n", "00")
	if err != ErrStaleTimestamp {
		t.Fatalf("want ErrStaleTimestamp, got %v", err)
	}
}

func TestVerifyRejectsBadTimestamp(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/open/v1/ping", nil)
	err := VerifyRequest(&AuthContext{Secret: "s"}, r, nil, "op_x", "not-a-number", "n", "00")
	if err == nil || !strings.Contains(err.Error(), "timestamp") {
		t.Fatalf("want timestamp error, got %v", err)
	}
}

func TestSignPayloadDeterministic(t *testing.T) {
	a := SignPayload("s", "123", []byte("payload"))
	b := SignPayload("s", "123", []byte("payload"))
	if a != b || len(a) != 64 {
		t.Fatalf("payload signature not deterministic hex: %q vs %q", a, b)
	}
	if SignPayload("s", "124", []byte("payload")) == a {
		t.Fatal("timestamp must be part of payload signature")
	}
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
