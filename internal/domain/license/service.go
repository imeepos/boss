package license

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Config 授权域配置(环境变量注入,见 config.Config.License)。
type Config struct {
	// PublicKeyHex release-platform 离线签发公钥(hex,32 字节 ed25519)。
	PublicKeyHex string
	// ProductID release-platform 产品 ID(激活兑码用)。
	ProductID string
	// CertPath 本地证书文件路径。
	CertPath string
	// DeviceID 本部署实例标识(如 hostname)。
	DeviceID string
	// Fingerprint 机器指纹(防证书复制;激活时上报 release-platform)。
	Fingerprint string
	// Leeway 时钟偏差容忍(默认 5m)。
	Leeway time.Duration
	// APIBaseURL release-platform API 地址(激活/取令牌用,如 http://192.168.0.102:38080)。
	APIBaseURL string
	// APIToken release-platform API token(rpat_ 开头;激活/取令牌需 bearer)。
	APIToken string
}

// Activator 激活链路抽象(兑码→取离线令牌);APIClient 实现,测试可注入。
type Activator interface {
	Exchange(ctx context.Context, activationCode, productID, deviceID, fingerprint string) (*Claims, []byte, error)
}

// Service 授权门禁服务:持有验签器+存储,对外提供 Check/Status/Activate。
type Service struct {
	Verifier *Verifier
	Store    *Store
	Cfg      Config
	// FingerprintFn 机器指纹计算函数(可测注入;nil 用配置值)。
	FingerprintFn func() string
	// API 激活客户端(可空;nil 时激活接口返回 not configured)。
	API Activator
}

// Check 校验本地证书:返回当前授权状态。每次请求都调用(门禁中间件);
// 验签成本极低(单次 ed25519),无缓存。
func (s *Service) Check(ctx context.Context) (Status, error) {
	data, err := s.Store.Load(ctx)
	if err != nil {
		return Status{}, err
	}
	claims, err := s.Verifier.Verify(data, VerifyOptions{
		Now:                 time.Now().UTC(),
		Leeway:              s.leeway(),
		ExpectedDeviceID:    s.Cfg.DeviceID,
		ExpectedFingerprint: s.fingerprint(),
	})
	if err != nil {
		return Status{}, err
	}
	return s.statusFromClaims(claims), nil
}

// Status 授权状态(供 admin 页/门禁展示)。
type Status struct {
	Activated   bool      `json:"activated"`
	LicenseID   string    `json:"licenseId,omitempty"`
	ProductID   string    `json:"productId,omitempty"`
	LicenseType string    `json:"licenseType,omitempty"`
	DeviceID    string    `json:"deviceId,omitempty"`
	Fingerprint string    `json:"fingerprint,omitempty"`
	ExpiresAt   *string   `json:"expiresAt,omitempty"`
	GraceEndsAt *string   `json:"graceEndsAt,omitempty"`
	InGrace     bool      `json:"inGrace"`
	CheckedAt   time.Time `json:"checkedAt"`
}

func (s *Service) statusFromClaims(c *Claims) Status {
	st := Status{
		Activated:   true,
		LicenseID:   c.LicenseID,
		ProductID:   c.ProductID,
		LicenseType: c.LicenseType,
		DeviceID:    c.DeviceID,
		Fingerprint: c.FingerprintHash,
		CheckedAt:   time.Now().UTC(),
	}
	if c.ExpiresAt != nil {
		v := c.ExpiresAt.Format(time.RFC3339)
		st.ExpiresAt = &v
	}
	if c.GraceEndsAt != nil {
		v := c.GraceEndsAt.Format(time.RFC3339)
		st.GraceEndsAt = &v
	}
	if c.ExpiresAt != nil && time.Now().UTC().After(*c.ExpiresAt) {
		st.InGrace = true
	}
	return st
}

// Activate 激活:调 release-platform 激活码兑换 → 取离线令牌 → 本地落盘 → 校验。
func (s *Service) Activate(ctx context.Context, activationCode string) (Status, error) {
	if s.API == nil {
		return Status{}, errors.New("license: activation client not configured")
	}
	claims, tokenData, err := s.API.Exchange(ctx, activationCode, s.Cfg.ProductID, s.Cfg.DeviceID, s.fingerprint())
	if err != nil {
		return Status{}, err
	}
	if err := s.Store.Save(ctx, tokenData); err != nil {
		return Status{}, err
	}
	data, err := s.Store.Load(ctx)
	if err != nil {
		return Status{}, err
	}
	if _, err := s.Verifier.Verify(data, VerifyOptions{
		Now:                 time.Now().UTC(),
		Leeway:              s.leeway(),
		ExpectedDeviceID:    s.Cfg.DeviceID,
		ExpectedFingerprint: s.fingerprint(),
	}); err != nil {
		return Status{}, fmt.Errorf("license: activate %w", err)
	}
	return s.statusFromClaims(claims), nil
}

// leeway 时钟容忍,缺省 5 分钟。
func (s *Service) leeway() time.Duration {
	if s.Cfg.Leeway > 0 {
		return s.Cfg.Leeway
	}
	return 5 * time.Minute
}

// fingerprint 取当前机器指纹(FingerprintFn 优先,配置兜底)。
func (s *Service) fingerprint() string {
	if s.FingerprintFn != nil {
		if v := s.FingerprintFn(); v != "" {
			return v
		}
	}
	return s.Cfg.Fingerprint
}