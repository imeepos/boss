// Package credential LOID 接入凭据编解码与校验(PAP 明文比对 / CHAP 按 RFC 1994)。
// 落库形态:v1$gcm$<nonce-b64>$<ct-b64>,AES-256-GCM 密文、密钥外置,库内无明文;
// CHAP 响应=MD5(ident||password||challenge)要求服务端可还原口令,单工哈希不可实现,
// 存储方案裁定:docs/notes/adopted/2026-09-06-aaa-credential-storage.md。
package credential

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5" // RFC 1994 规定 CHAP 摘要算法,非通用散列用途
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// formatPrefix 落库凭据代际前缀;未来换算法时新旧代际可并存判别。
const formatPrefix = "v1$gcm$"

// ErrMalformed 凭据串损坏或代际不识别(按校验失败处理,不 panic)。
var ErrMalformed = errors.New("credential: malformed stored credential")

// Codec 凭据编解码器:密钥材料经 SHA-256 归一为 AES-256 密钥。
type Codec struct {
	aead cipher.AEAD
}

// New 构造;material 为密钥材料(推荐 32 字节随机 hex;任意非空串皆可)。
func New(material string) (*Codec, error) {
	if material == "" {
		return nil, errors.New("credential: empty key material")
	}
	sum := sha256.Sum256([]byte(material))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, fmt.Errorf("credential: new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("credential: new gcm: %w", err)
	}
	return &Codec{aead: aead}, nil
}

// Encode 明文口令编码为落库凭据串(每次随机 nonce:同口令密文不同,不可反查等值)。
func (c *Codec) Encode(password string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("credential: read nonce: %w", err)
	}
	ct := c.aead.Seal(nil, nonce, []byte(password), nil)
	return formatPrefix +
		base64.RawStdEncoding.EncodeToString(nonce) + "$" +
		base64.RawStdEncoding.EncodeToString(ct), nil
}

// decode 还原口令明文(仅 CHAP 路径需要;损坏/异代际一律 ErrMalformed)。
func (c *Codec) decode(stored string) (string, error) {
	rest, ok := strings.CutPrefix(stored, formatPrefix)
	if !ok {
		return "", ErrMalformed
	}
	parts := strings.Split(rest, "$")
	if len(parts) != 2 {
		return "", ErrMalformed
	}
	nonce, errNonce := base64.RawStdEncoding.DecodeString(parts[0])
	ct, errCt := base64.RawStdEncoding.DecodeString(parts[1])
	if errNonce != nil || errCt != nil || len(nonce) != c.aead.NonceSize() {
		return "", ErrMalformed
	}
	plain, err := c.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", ErrMalformed
	}
	return string(plain), nil
}

// Equal PAP 校验:常数时间比对落库口令与请求口令。
func (c *Codec) Equal(stored, password string) bool {
	plain, err := c.decode(stored)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(plain), []byte(password)) == 1
}

// CHAPOK RFC 1994 校验:resp == MD5(ident||password||challenge),常数时间比对。
func (c *Codec) CHAPOK(stored string, ident byte, challenge, resp []byte) bool {
	plain, err := c.decode(stored)
	if err != nil {
		return false
	}
	h := md5.New()
	h.Write([]byte{ident})
	h.Write([]byte(plain))
	h.Write(challenge)
	return subtle.ConstantTimeCompare(h.Sum(nil), resp) == 1
}

// randomAlphabet 随机口令字符集(去易混字符 i/l/o/I/L/O/0/1,口头转抄不失真)。
const randomAlphabet = "abcdefghjkmnpqrstuvwxyzACDEFGHJKMNPQRSTUVWXYZ23456789"

// Random 生成 n 位随机口令(crypto/rand);n<8 按 8 处理。
func Random(n int) (string, error) {
	if n < 8 {
		n = 8
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("credential: random: %w", err)
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = randomAlphabet[int(b)%len(randomAlphabet)]
	}
	return string(out), nil
}

// NewResolved 按配置装配编解码器:credKey(BOSS_AAA_CRED_KEY/FILE)显式优先;
// 空则从 NAS Secret 派生(开发兜底)。derived=true 表示走了派生路径,调用方必须
// 输出 [aaa] CRED KEY ALERT 留痕(生产必须显式配置独立密钥)。派生式与既有
// cmd/aaa 口径逐字一致,否则历史密文不可解。
func NewResolved(credKey, secret string) (*Codec, bool, error) {
	material := credKey
	derived := false
	if material == "" {
		material = "boss-aaa-cred-key|" + secret
		derived = true
	}
	c, err := New(material)
	return c, derived, err
}
