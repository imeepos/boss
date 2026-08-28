package app

import (
	"path/filepath"
	"testing"

	"github.com/ymm-001/boss/internal/domain/license"
	"github.com/ymm-001/boss/internal/pkg/buildinfo"
	"github.com/ymm-001/boss/internal/pkg/config"
)

// pubHex 测试公钥(与 102 release-platform 签发密钥同源,仅验签格式)。
const pubHex = "f82d2db7a72efc66f8ffca0a67a7f8a29c6c6ea8f720fe5c139afdaa8869ff63"

// TestWireLicenseServiceNoEmbeddedKey 未内嵌公钥 = 开发态:返回 nil(门禁放行)。
func TestWireLicenseServiceNoEmbeddedKey(t *testing.T) {
	orig := buildinfo.LicensePublicKeyHex
	buildinfo.LicensePublicKeyHex = ""
	defer func() { buildinfo.LicensePublicKeyHex = orig }()

	svc := wireLicenseService(&config.Config{})
	if svc != nil {
		t.Fatalf("no embedded key should yield nil service, got %+v", svc)
	}
}

// TestWireLicenseServiceWithEmbeddedKey 内嵌合法公钥 = 强制门禁:服务装配齐全。
func TestWireLicenseServiceWithEmbeddedKey(t *testing.T) {
	orig := buildinfo.LicensePublicKeyHex
	buildinfo.LicensePublicKeyHex = pubHex
	defer func() { buildinfo.LicensePublicKeyHex = orig }()

	cfg := &config.Config{}
	cfg.License.CertPath = filepath.Join(t.TempDir(), "license.json")
	cfg.License.DeviceID = "test-host"
	svc := wireLicenseService(cfg)
	if svc == nil {
		t.Fatal("embedded key should yield service")
	}
	if svc.Verifier == nil || svc.Store == nil {
		t.Fatal("verifier/store must be wired")
	}
	if svc.Cfg.DeviceID != "test-host" {
		t.Fatalf("device id not wired: %s", svc.Cfg.DeviceID)
	}
}

// TestWireLicenseServiceBadEmbeddedKey 内嵌非法公钥 = 门禁不装配(ALERT 留痕),返回 nil。
func TestWireLicenseServiceBadEmbeddedKey(t *testing.T) {
	orig := buildinfo.LicensePublicKeyHex
	buildinfo.LicensePublicKeyHex = "not-a-hex-key"
	defer func() { buildinfo.LicensePublicKeyHex = orig }()

	svc := wireLicenseService(&config.Config{})
	if svc != nil {
		t.Fatalf("bad embedded key should disable gate (nil), got %+v", svc)
	}
}

// TestStoreNoCertYet 强制门禁下无证书文件 = ErrNoLicense(启动阶段的预期状态)。
func TestStoreNoCertYet(t *testing.T) {
	orig := buildinfo.LicensePublicKeyHex
	buildinfo.LicensePublicKeyHex = pubHex
	defer func() { buildinfo.LicensePublicKeyHex = orig }()

	cfg := &config.Config{}
	cfg.License.CertPath = filepath.Join(t.TempDir(), "license.json")
	svc := wireLicenseService(cfg)
	_, err := svc.Check(t.Context())
	if err != license.ErrNoLicense {
		t.Fatalf("no cert file should yield ErrNoLicense, got %v", err)
	}
}
