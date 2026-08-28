package license

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

// TestExchangeFullChain 回归:Exchange 必须只把令牌本体 {payload,key_id,signature,algorithm}
// 交给 Verify 严格解析——曾因序列化整个 response(多出 license_id/expires_at)被
// DisallowUnknownFields 拒绝,激活全链路 502(2026-08-27 实测)。
func TestExchangeFullChain(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	pubHex := hex.EncodeToString(pub)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/activations", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"license_id": "ac-x1"})
	})
	mux.HandleFunc("/v1/licenses/ac-x1/offline-token", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().UTC()
		claims := Claims{
			LicenseID: "ac-x1", ProductID: "boss-server",
			DeviceID: "dev-9", FingerprintHash: "fp-9",
			LicenseType: "duration", Status: "consumed",
			ExpiresAt: ptrTime(now.Add(24 * time.Hour)),
			IssuedAt:  now, NotBefore: now,
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"license_id":     "ac-x1",
			"token":          json.RawMessage(signTokenBytes(priv, claims)),
			"public_key_hex": pubHex,
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewAPIClient(srv.URL, "rpat_test", "", pubHex)
	svc := &Service{
		Verifier: mustVerifierPub(t, pubHex),
		Store:    &Store{Path: filepath.Join(t.TempDir(), "l.json")},
		Cfg:      Config{DeviceID: "dev-9", Fingerprint: "fp-9"},
		API:      c,
	}
	st, err := svc.Activate(context.Background(), "CODE-1")
	if err != nil {
		t.Fatalf("Activate full chain: %v", err)
	}
	if !st.Activated || st.LicenseID != "ac-x1" {
		t.Fatalf("unexpected status %+v", st)
	}
	// 落盘内容必须是纯令牌四字段(无外层 response 字段),否则 Check 会解析失败。
	raw, err := svc.Store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var probe map[string]any
	if err := json.Unmarshal(raw, &probe); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"payload", "signature", "algorithm"} {
		if _, ok := probe[k]; !ok {
			t.Fatalf("stored cert missing %q: %v", k, probe)
		}
	}
	if _, ok := probe["license_id"]; ok {
		t.Fatal("stored cert must not contain outer fields")
	}
}

func mustVerifierPub(t *testing.T, pubHex string) *Verifier {
	t.Helper()
	v, err := NewVerifier(pubHex)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
