// Package license 系统级授权门禁域:boss 整体功能受 release-platform 授权控制。
//
// 离线授权模式(裁定点):release-platform 签发 Ed25519 签名的离线令牌
// (payload=base64(JSON claims),signature=hex(ed25519))。boss 内嵌公钥
// 本地验签,不依赖网络;证书绑定机器指纹防复制。见 adopted 授权对接 note。
package license

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

// ed25519SignatureSize 固定 64 字节(与 crypto/ed25519 常量一致)。
const ed25519SignatureSize = ed25519.SignatureSize

// 令牌中的 claims 字段(与 release-platform api.offlineLicenseClaims 对齐)。
type Claims struct {
	LicenseID       string     `json:"license_id"`
	ProductID       string     `json:"product_id"`
	DeviceID        string     `json:"device_id"`
	FingerprintHash string     `json:"fingerprint_hash"`
	LicenseType     string     `json:"license_type"` // duration|lifetime|trial
	Status          string     `json:"status"`
	Binding         Binding    `json:"binding,omitempty"` // 激活时钉的设备维度
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	GraceEndsAt     *time.Time `json:"grace_ends_at,omitempty"`
	IssuedAt        time.Time  `json:"issued_at"`
	NotBefore       time.Time  `json:"not_before"`
}

// Binding release-platform DeviceBinding 投影(可选多维度:MAC/IP/域名/machine-id/复合指纹)。
type Binding struct {
	MACs      []string `json:"macs,omitempty"`
	IPs       []string `json:"ips,omitempty"`
	Domain    string   `json:"domain,omitempty"`
	MachineID string   `json:"machine_id,omitempty"`
	Composite string   `json:"composite,omitempty"`
}

// Token 是 release-platform 签发的离线令牌原文(字段与 api.offlineLicenseToken 对齐)。
type Token struct {
	Payload   string `json:"payload"`   // base64(claims JSON bytes)
	KeyID     string `json:"key_id"`
	Signature string `json:"signature"` // hex Ed25519,签的是解码后的 payload 字节
	Algorithm string `json:"algorithm"` // ed25519
}

var (
	// ErrInvalidToken 令牌结构解析失败(未知字段/缺字段/非法 base64/hex)。
	ErrInvalidToken = errors.New("license: invalid token")
	// ErrBadSignature 验签失败(公钥不匹配或内容被篡改)。
	ErrBadSignature = errors.New("license: bad signature")
	// ErrWrongDevice 证书绑定的设备与当前机器不符(防复制)。
	ErrWrongDevice = errors.New("license: bound to different device")
	// ErrNotYetValid 证书未到生效时间。
	ErrNotYetValid = errors.New("license: not yet valid")
	// ErrExpired 证书已过有效期(含宽限仍过)。
	ErrExpired = errors.New("license: expired")
	// ErrRevoked 证书状态为 revoked(服务端吊销)。
	ErrRevoked = errors.New("license: revoked")
	// ErrNoLicense 本地没有证书文件。
	ErrNoLicense = errors.New("license: not activated")
)

// parseToken 严格解码令牌:未知字段拒绝,payload/signature 必须合法。
func parseToken(data []byte) (*Token, []byte, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var t Token
	if err := dec.Decode(&t); err != nil {
		return nil, nil, err
	}
	if t.Algorithm != "ed25519" {
		return nil, nil, errors.New("unsupported algorithm")
	}
	payload, err := base64.StdEncoding.DecodeString(t.Payload)
	if err != nil {
		return nil, nil, err
	}
	sig, err := hex.DecodeString(t.Signature)
	if err != nil {
		return nil, nil, err
	}
	if len(sig) != ed25519SignatureSize {
		return nil, nil, errors.New("bad signature length")
	}
	if len(payload) == 0 {
		return nil, nil, errors.New("empty payload")
	}
	return &t, payload, nil
}

// parseClaims 解码 claims;未知字段拒绝。
func parseClaims(payload []byte) (*Claims, error) {
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	var c Claims
	if err := dec.Decode(&c); err != nil {
		return nil, err
	}
	if c.LicenseID == "" || c.DeviceID == "" || c.FingerprintHash == "" || c.IssuedAt.IsZero() {
		return nil, errors.New("license claims incomplete")
	}
	return &c, nil
}