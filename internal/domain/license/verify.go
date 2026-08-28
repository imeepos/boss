package license

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"time"
)

// Verifier 离线授权验签内核:持有内嵌公钥,校验签名+绑定+时间窗+状态。
// 线程安全(公钥只读);now 由外部注入以保证可测。
type Verifier struct {
	pub ed25519.PublicKey
}

// NewVerifier 由 hex 公钥构造验签器(32 字节 ed25519 公钥)。
func NewVerifier(pubHex string) (*Verifier, error) {
	pub, err := hex.DecodeString(pubHex)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: bad public key hex", ErrInvalidToken)
	}
	return &Verifier{pub: pub}, nil
}

// VerifyOptions 校验上下文:now 缺省为当前 UTC,leeway 容忍时钟偏差,
// ExpectedDevice/ExpectedFingerprint 与 claims 绑定维度比对(空则跳过)。
type VerifyOptions struct {
	Now                 time.Time
	Leeway              time.Duration
	ExpectedDeviceID    string
	ExpectedFingerprint string
}

// Verify 完整校验一枚离线令牌:签名 → 结构 → 绑定 → 时间窗 → 状态。
// 任何一步失败返回对应哨兵错误(可 grep 的明确原因,不静默)。
func (v *Verifier) Verify(tokenData []byte, opts VerifyOptions) (*Claims, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tok, payload, err := parseToken(tokenData)
	if err != nil {
		return nil, fmt.Errorf("%w: parse: %v", ErrInvalidToken, err)
	}
	if !ed25519.Verify(v.pub, payload, mustSig(tok)) {
		return nil, ErrBadSignature
	}
	claims, err := parseClaims(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: claims: %v", ErrInvalidToken, err)
	}
	if err := checkBindings(claims, opts); err != nil {
		return nil, err
	}
	if err := checkTime(claims, now, opts.Leeway); err != nil {
		return nil, err
	}
	if err := checkStatus(claims); err != nil {
		return nil, err
	}
	return claims, nil
}

// mustSig 取 token 里已合法解码的签名(parseToken 保证非空且 64 字节)。
func mustSig(t *Token) []byte {
	sig, _ := hex.DecodeString(t.Signature)
	return sig
}

// checkBindings 设备/指纹绑定;两维度都非空时才强制匹配。
func checkBindings(c *Claims, opts VerifyOptions) error {
	if opts.ExpectedDeviceID != "" && opts.ExpectedDeviceID != c.DeviceID {
		return ErrWrongDevice
	}
	if opts.ExpectedFingerprint != "" && opts.ExpectedFingerprint != c.FingerprintHash {
		return ErrWrongDevice
	}
	return nil
}

// checkTime 时间窗:NotBefore 之前或 ExpiresAt(+leeway) 之后拒绝;
// 宽限期(grace)视为有限续期——若在宽限内则放行(业务侧可降级提示)。
func checkTime(c *Claims, now time.Time, leeway time.Duration) error {
	if !c.NotBefore.IsZero() && now.Before(c.NotBefore.Add(-leeway)) {
		return ErrNotYetValid
	}
	if c.ExpiresAt != nil && now.After(c.ExpiresAt.Add(leeway)) {
		if c.GraceEndsAt != nil && now.Before(c.GraceEndsAt.Add(leeway)) {
			return nil // 宽限期内仍可用,业务侧标记降级
		}
		return ErrExpired
	}
	return nil
}

// checkStatus 服务端吊销状态。
func checkStatus(c *Claims) error {
	if c.Status == "revoked" || c.Status == "expired" {
		return ErrRevoked
	}
	return nil
}
