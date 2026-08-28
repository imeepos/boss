package license

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"path/filepath"
	"testing"
)

// newTestService 组装可测 Service:真实验签器+临时文件存储。
func newTestService(t *testing.T, priv ed25519.PrivateKey) *Service {
	t.Helper()
	pub := priv.Public().(ed25519.PublicKey)
	v, err := NewVerifier(hex.EncodeToString(pub))
	if err != nil {
		t.Fatal(err)
	}
	return &Service{
		Verifier: v,
		Store:    &Store{Path: filepath.Join(t.TempDir(), "license.json")},
		Cfg: Config{
			DeviceID:    "boss-license-dev-01",
			Fingerprint: "fp-boss-01",
		},
	}
}

func TestCheckNoLicense(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	s := newTestService(t, priv)
	_, err := s.Check(context.Background())
	if !errors.Is(err, ErrNoLicense) {
		t.Fatalf("want ErrNoLicense, got %v", err)
	}
}

func TestCheckAfterActivate(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	s := newTestService(t, priv)
	tok := signToken(t, priv, fixtureClaims())
	if err := s.Store.Save(context.Background(), tok); err != nil {
		t.Fatal(err)
	}
	st, err := s.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !st.Activated || st.LicenseID != "ac-test-1" {
		t.Fatalf("unexpected status: %+v", st)
	}
}

func TestActivateSavesAndVerifies(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	s := newTestService(t, priv)
	s.API = fakeActivator{priv: priv}
	st, err := s.Activate(context.Background(), "TEST-CODE-1")
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if !st.Activated {
		t.Fatalf("want activated, got %+v", st)
	}
	st2, err := s.Check(context.Background())
	if err != nil {
		t.Fatalf("Check after activate: %v", err)
	}
	if st2.LicenseID != "ac-test-1" {
		t.Fatalf("license id mismatch: %+v", st2)
	}
}

func TestActivateUnconfigured(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	s := newTestService(t, priv)
	_, err := s.Activate(context.Background(), "TEST-CODE")
	if err == nil {
		t.Fatal("want error when activator missing")
	}
}

func TestActivateWrongDeviceRejected(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	s := newTestService(t, priv)
	// 假激活器返回绑定到"别的机器"的证书:claims.DeviceID 固定 other-host。
	s.API = fakeActivator{priv: priv, boundDevice: "other-host", boundFP: "fp-other"}
	_, err := s.Activate(context.Background(), "TEST-CODE-2")
	if !errors.Is(err, ErrWrongDevice) {
		t.Fatalf("want ErrWrongDevice, got %v", err)
	}
}

// fakeActivator 假激活器:模拟 release-platform 给 boundDevice/boundFP 机器签发令牌。
type fakeActivator struct {
	priv        ed25519.PrivateKey
	boundDevice string // 空=按请求参数签发(跟随调用方)
	boundFP     string
}

func (f fakeActivator) Exchange(_ context.Context, code, productID, deviceID, fp string) (*Claims, []byte, error) {
	c := fixtureClaims()
	// 模拟远端已有绑定:boundDevice 非空时无视本次请求参数,签发绑定旧设备。
	if f.boundDevice != "" {
		c.DeviceID = f.boundDevice
		c.FingerprintHash = f.boundFP
	} else {
		c.DeviceID = deviceID
		c.FingerprintHash = fp
	}
	c.ProductID = productID
	return claimsPtr(c), signTokenBytes(f.priv, c), nil
}

func claimsPtr(c Claims) *Claims { return &c }
