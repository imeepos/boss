package app

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ymm-001/boss/internal/domain/license"
	"github.com/ymm-001/boss/internal/pkg/config"
)

// wireLicenseService 装配系统级授权门禁服务。
//
// 语义:BOSS_LICENSE_ENABLED=true 时启用——业务接口须持有 release-platform
// 签发的有效离线证书(本地 Ed25519 验签,不依赖网络);未启用返回 nil(完全放行)。
// 启用但缺公钥/API 配置视为配置错误,启动即失败(显式失败优于静默放行)。
func wireLicenseService(cfg *config.Config) *license.Service {
	lc := cfg.License
	if !lc.Enabled {
		return nil
	}
	if lc.PublicKeyHex == "" {
		log.Fatalf("[license] ALERT: BOSS_LICENSE_ENABLED=true but BOSS_LICENSE_PUBLIC_KEY_HEX empty")
	}
	verifier, err := license.NewVerifier(lc.PublicKeyHex)
	if err != nil {
		log.Fatalf("[license] ALERT: bad public key: %v", err)
	}
	svc := &license.Service{
		Verifier: verifier,
		Store:    &license.Store{Path: lc.CertPath},
		Cfg: license.Config{
			PublicKeyHex: lc.PublicKeyHex,
			ProductID:    lc.ProductID,
			CertPath:     lc.CertPath,
			DeviceID:     lc.DeviceID,
			Fingerprint:  lc.Fingerprint,
			Leeway:       time.Duration(lc.LeewayMinutes) * time.Minute,
			APIBaseURL:   lc.APIBaseURL,
			APIToken:     lc.APIToken,
		},
	}
	// 指纹函数:配置优先,否则回退机器指纹(实现注入;失败返回配置值)。
	svc.FingerprintFn = func() string {
		if v := machineFingerprint(); v != "" {
			return v
		}
		return lc.Fingerprint
	}
	// 激活客户端:仅当 API 地址与 token 齐备时接入;否则激活端点报 not configured。
	if lc.APIBaseURL != "" && lc.APIToken != "" {
		svc.API = license.NewAPIClient(lc.APIBaseURL, lc.APIToken, "", lc.PublicKeyHex)
	} else {
		log.Printf("[license] WARN: API 配置缺失,激活端点不可用(仅验签门禁生效)")
	}
	return svc
}

// machineFingerprint 当前机器指纹:优先 /etc/machine-id,其次 hostname 哈希。
// 指纹与激活时上报值一致(防证书复制);取不到返回空(用配置兜底)。
func machineFingerprint() string {
	if data, err := os.ReadFile("/etc/machine-id"); err == nil {
		if s := strings.TrimSpace(string(data)); s != "" {
			return s
		}
	}
	host, err := os.Hostname()
	if err != nil || host == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(host))
	return hex.EncodeToString(sum[:])
}