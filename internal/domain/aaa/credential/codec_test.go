package credential

import (
	"bytes"
	"crypto/md5"
	"strings"
	"testing"
)

func mustCodec(t *testing.T) *Codec {
	t.Helper()
	c, err := New("test-key-material-hex")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestCodec_EncodeRoundtrip(t *testing.T) {
	c := mustCodec(t)
	stored, err := c.Encode("s3cret-PW")
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !strings.HasPrefix(stored, "v1$gcm$") {
		t.Fatalf("stored=%s want v1$gcm$ prefix", stored)
	}
	if strings.Contains(stored, "s3cret-PW") {
		t.Fatal("stored credential leaks plaintext")
	}
	if !c.Equal(stored, "s3cret-PW") {
		t.Fatal("Equal(correct)=false")
	}
	if c.Equal(stored, "wrong") {
		t.Fatal("Equal(wrong)=true")
	}
}

func TestCodec_EncodeNonDeterministic(t *testing.T) {
	c := mustCodec(t)
	a, _ := c.Encode("same-pw")
	b, _ := c.Encode("same-pw")
	if a == b {
		t.Fatal("same password must yield distinct ciphertext (nonce)")
	}
}

func TestCodec_MalformedStored(t *testing.T) {
	c := mustCodec(t)
	for _, stored := range []string{"", "plaintext", "v1$gcm$", "v1$gcm$abc", "v1$gcm$!!!$!!!"} {
		if c.Equal(stored, "x") {
			t.Fatalf("Equal(%q)=true, want false", stored)
		}
	}
}

func TestCodec_CHAPOK(t *testing.T) {
	c := mustCodec(t)
	stored, _ := c.Encode("chap-pw")
	ident := byte(5)
	challenge := []byte{0x01, 0x02, 0x03, 0x04}

	h := md5.New()
	h.Write([]byte{ident})
	h.Write([]byte("chap-pw"))
	h.Write(challenge)
	good := h.Sum(nil)

	if !c.CHAPOK(stored, ident, challenge, good) {
		t.Fatal("CHAPOK(correct)=false")
	}
	bad := bytes.Clone(good)
	bad[0] ^= 0xFF
	if c.CHAPOK(stored, ident, challenge, bad) {
		t.Fatal("CHAPOK(tampered)=true")
	}
	if c.CHAPOK(stored, ident, []byte("other-challenge"), good) {
		t.Fatal("CHAPOK(wrong challenge)=true")
	}
	if c.CHAPOK("v1$gcm$broken", ident, challenge, good) {
		t.Fatal("CHAPOK(malformed)=true")
	}
}

func TestCodec_NewEmptyMaterial(t *testing.T) {
	if _, err := New(""); err == nil {
		t.Fatal("empty key material must be rejected")
	}
}

func TestRandom(t *testing.T) {
	pw, err := Random(16)
	if err != nil {
		t.Fatalf("Random: %v", err)
	}
	if len(pw) != 16 {
		t.Fatalf("len=%d want 16", len(pw))
	}
	for _, ch := range pw {
		if !strings.ContainsRune(randomAlphabet, ch) {
			t.Fatalf("unexpected char %q", ch)
		}
	}
	short, _ := Random(3)
	if len(short) != 8 {
		t.Fatalf("min-length clamp broken: %d", len(short))
	}
	other, _ := Random(16)
	if pw == other {
		t.Fatal("two random passwords identical")
	}
}

func TestNewResolved(t *testing.T) {
	explicit, derived, err := NewResolved("hex-key-a", "nas-secret")
	if err != nil || derived {
		t.Fatalf("explicit: derived=%v err=%v", derived, err)
	}
	direct, _ := New("hex-key-a")
	stored, _ := direct.Encode("pw-1234")
	if !explicit.Equal(stored, "pw-1234") {
		t.Fatal("explicit codec mismatch with New(same material)")
	}
	fallback, derived, err := NewResolved("", "nas-secret")
	if err != nil || !derived {
		t.Fatalf("fallback: derived=%v err=%v", derived, err)
	}
	want, _ := New("boss-aaa-cred-key|nas-secret")
	stored2, _ := want.Encode("pw-5678")
	if !fallback.Equal(stored2, "pw-5678") {
		t.Fatal("fallback material derivation changed")
	}
	bothEmpty, derived, err := NewResolved("", "")
	if err != nil || !derived {
		t.Fatalf("both empty: derived=%v err=%v", derived, err)
	}
	if bothEmpty == nil {
		t.Fatal("both empty codec nil")
	}
}
