package app

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ymm-001/boss/internal/domain/license"
	"github.com/ymm-001/boss/internal/pkg/buildinfo"
	"github.com/ymm-001/boss/internal/pkg/config"
)

// wireLicenseService 装配系统级授权门禁服务(B 档强制门禁)。
//
// 语义:公钥**编译期注入**(buildinfo.LicensePublicKeyHex,经 -ldflags -X)。
//   - 已注入公钥 → 强制门禁:业务接口须持有 release-platform 签发的有效离线证书,
//     本地 Ed25519 验签;无证书/失效 → 403 LICENSE_REQUIRED(激活页仍可达)。
//   - 未注入公钥 → 开发构建:门禁不启用,打 ALERT 日志(明确"未内嵌授权公钥"状态),
//     业务完全放行,便于无证书环境开发调试。
//
// 无"环境变量开关"可关(B 档):绕过门禁的唯一途径是重新编译替换二进制,
// 而非改配置文件。裁定见 adopted license-gate note(Amended)。
func wireLicenseService(cfg *config.Config) *license.Service {
	if buildinfo.LicensePublicKeyHex == "" {
		log.Printf("[license] ALERT: 未内嵌授权公钥(buildinfo.LicensePublicKeyHex 为空),门禁未启用——业务放行;生产构建必须注入公钥")
		return nil
	}
	verifier, err := license.NewVerifier(buildinfo.LicensePublicKeyHex)
	if err != nil {
		// 内嵌公钥非法 = 构建产物损坏:不装配门禁并打 ALERT(门禁缺位等同放行,
		// 但日志明确记录原因,避免静默);正常构建链路不会产出此状态。
		log.Printf("[license] ALERT: bad embedded public key (%v), gate disabled", err)
		return nil
	}
	lc := cfg.License
	svc := &license.Service{
		Verifier: verifier,
		Store:    &license.Store{Path: lc.CertPath},
		Cfg: license.Config{
			PublicKeyHex: buildinfo.LicensePublicKeyHex,
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
		svc.API = license.NewAPIClient(lc.APIBaseURL, lc.APIToken, "", buildinfo.LicensePublicKeyHex)
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
