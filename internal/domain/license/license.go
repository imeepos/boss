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
	"fmt"
	"time"
)

// ed25519SignatureSize 固定 64 字节(与 crypto/ed25519 常量一致)。
const ed25519SignatureSize = ed25519.SignatureSize

// 外部契约编解码红线(对齐 internal/pkg/stripe/source.go 先例):
// release-platform 载荷为 snake_case,外部字段名不经 struct tag,
// 一律经自定义 Marshal/Unmarshal 的 map 双向映射。Claims/Binding/Token
// 因此不携 json tag(jsontags 门禁对无 tag 字段跳过),JSON 形态保持
// snake_case 与原契约完全一致。

// Claims 令牌中的 claims 字段(与 release-platform api.offlineLicenseClaims 对齐)。
type Claims struct {
	LicenseID       string
	ProductID       string
	DeviceID        string
	FingerprintHash string
	LicenseType     string // duration|lifetime|trial
	Status          string
	Binding         Binding // 激活时钉的设备维度
	ExpiresAt       *time.Time
	GraceEndsAt     *time.Time
	IssuedAt        time.Time
	NotBefore       time.Time
}

// claimsRawKeys 白名单,未知键拒绝(等价原 DisallowUnknownFields 严格性)。
var claimsRawKeys = map[string]bool{
	"license_id": true, "product_id": true, "device_id": true,
	"fingerprint_hash": true, "license_type": true, "status": true,
	"binding": true, "expires_at": true, "grace_ends_at": true,
	"issued_at": true, "not_before": true,
}

// UnmarshalJSON Claims:未知键拒绝 + 逐字段映射(外部 snake_case 载荷)。
func (c *Claims) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	for k := range raw {
		if !claimsRawKeys[k] {
			return fmt.Errorf("license: unknown claims key %q", k)
		}
	}
	str := func(k string, dst *string) error {
		if v, ok := raw[k]; ok {
			return json.Unmarshal(v, dst)
		}
		return nil
	}
	for k, dst := range map[string]*string{
		"license_id": &c.LicenseID, "product_id": &c.ProductID, "device_id": &c.DeviceID,
		"fingerprint_hash": &c.FingerprintHash, "license_type": &c.LicenseType, "status": &c.Status,
	} {
		if err := str(k, dst); err != nil {
			return err
		}
	}
	timeOrNil := func(k string) (*time.Time, error) {
		v, ok := raw[k]
		if !ok {
			return nil, nil
		}
		var t time.Time
		if err := json.Unmarshal(v, &t); err != nil {
			return nil, err
		}
		return &t, nil
	}
	var err error
	if c.ExpiresAt, err = timeOrNil("expires_at"); err != nil {
		return err
	}
	if c.GraceEndsAt, err = timeOrNil("grace_ends_at"); err != nil {
		return err
	}
	if v, ok := raw["issued_at"]; ok {
		if err := json.Unmarshal(v, &c.IssuedAt); err != nil {
			return err
		}
	}
	if v, ok := raw["not_before"]; ok {
		if err := json.Unmarshal(v, &c.NotBefore); err != nil {
			return err
		}
	}
	if v, ok := raw["binding"]; ok {
		return json.Unmarshal(v, &c.Binding)
	}
	return nil
}

// MarshalJSON Claims:映射回 snake_case 载荷(密文/时间格式与 release-platform 契约一致)。
func (c Claims) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"license_id": c.LicenseID, "product_id": c.ProductID, "device_id": c.DeviceID,
		"fingerprint_hash": c.FingerprintHash, "license_type": c.LicenseType,
		"status": c.Status, "issued_at": c.IssuedAt, "not_before": c.NotBefore,
	}
	if c.Binding.MACs != nil || c.Binding.IPs != nil || c.Binding.Domain != "" ||
		c.Binding.MachineID != "" || c.Binding.Composite != "" {
		m["binding"] = c.Binding
	}
	if c.ExpiresAt != nil {
		m["expires_at"] = *c.ExpiresAt
	}
	if c.GraceEndsAt != nil {
		m["grace_ends_at"] = *c.GraceEndsAt
	}
	return json.Marshal(m)
}

// Binding release-platform DeviceBinding 投影(可选多维度:MAC/IP/域名/machine-id/复合指纹)。
type Binding struct {
	MACs      []string
	IPs       []string
	Domain    string
	MachineID string
	Composite string
}

var bindingRawKeys = map[string]bool{
	"macs": true, "ips": true, "domain": true, "machine_id": true, "composite": true,
}

// UnmarshalJSON Binding:未知键拒绝(同 Claims 严格性)。
func (b *Binding) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for k := range raw {
		if !bindingRawKeys[k] {
			return fmt.Errorf("license: unknown binding key %q", k)
		}
	}
	if v, ok := raw["macs"]; ok {
		_ = json.Unmarshal(v, &b.MACs)
	}
	if v, ok := raw["ips"]; ok {
		_ = json.Unmarshal(v, &b.IPs)
	}
	if v, ok := raw["domain"]; ok {
		_ = json.Unmarshal(v, &b.Domain)
	}
	if v, ok := raw["machine_id"]; ok {
		_ = json.Unmarshal(v, &b.MachineID)
	}
	if v, ok := raw["composite"]; ok {
		_ = json.Unmarshal(v, &b.Composite)
	}
	return nil
}

// MarshalJSON Binding:映射回 snake_case 载荷。
func (b Binding) MarshalJSON() ([]byte, error) {
	m := map[string]any{}
	if len(b.MACs) > 0 {
		m["macs"] = b.MACs
	}
	if len(b.IPs) > 0 {
		m["ips"] = b.IPs
	}
	if b.Domain != "" {
		m["domain"] = b.Domain
	}
	if b.MachineID != "" {
		m["machine_id"] = b.MachineID
	}
	if b.Composite != "" {
		m["composite"] = b.Composite
	}
	return json.Marshal(m)
}

// Token 是 release-platform 签发的离线令牌原文(字段与 api.offlineLicenseToken 对齐)。
type Token struct {
	Payload   string // base64(claims JSON bytes)
	KeyID     string
	Signature string // hex Ed25519,签的是解码后的 payload 字节
	Algorithm string // ed25519
}

var tokenRawKeys = map[string]bool{
	"payload": true, "key_id": true, "signature": true, "algorithm": true,
}

// UnmarshalJSON Token:未知键拒绝(与 parseToken 原 DisallowUnknownFields 等价)。
func (t *Token) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	for k := range raw {
		if !tokenRawKeys[k] {
			return fmt.Errorf("license: unknown token key %q", k)
		}
	}
	if v, ok := raw["payload"]; ok {
		_ = json.Unmarshal(v, &t.Payload)
	}
	if v, ok := raw["key_id"]; ok {
		_ = json.Unmarshal(v, &t.KeyID)
	}
	if v, ok := raw["signature"]; ok {
		_ = json.Unmarshal(v, &t.Signature)
	}
	if v, ok := raw["algorithm"]; ok {
		_ = json.Unmarshal(v, &t.Algorithm)
	}
	return nil
}

// MarshalJSON Token:映射回 snake_case 载荷(落盘格式与 release-platform 契约一致)。
func (t Token) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"payload": t.Payload, "key_id": t.KeyID,
		"signature": t.Signature, "algorithm": t.Algorithm,
	})
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
