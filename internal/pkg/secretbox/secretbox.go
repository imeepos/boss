// secretbox:认证配置 secret 字段的可逆加密(AES-256-GCM)。
// 密钥来源:BOSS_AUTH_SECRET_KEY 优先,回退 BOSS_JWT_SECRET(与进程现有 env 约定一致);
// 密文格式 enc:v1:<base64(nonce|ciphertext)>,非该前缀的值视为明文历史数据(兼容读取)。
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

const prefix = "enc:v1:"

// keyFn 密钥来源函数,可注入替换以便单测覆盖 newGCM 错误分支。
var keyFn = Key

// defaultKeyWarned 内置默认密钥只告警一次(Seal/Open 高频调用)。
var defaultKeyWarned sync.Once

// Key 从 env 派生 32 字节 AES 密钥。
func Key() []byte {
	secret := os.Getenv("BOSS_AUTH_SECRET_KEY")
	if secret == "" {
		secret = os.Getenv("BOSS_JWT_SECRET")
	}
	if secret == "" {
		defaultKeyWarned.Do(func() {
			log.Printf("[secretbox] WARNING 未设置 BOSS_AUTH_SECRET_KEY/BOSS_JWT_SECRET,使用内置默认密钥加密配置 secret;" +
				"换用正式密钥需先在后台重存各通道密钥再轮换环境变量")
		})
		secret = "boss-auth-secret"
	}
	k := sha256.Sum256([]byte(secret))
	return k[:]
}

// Seal 加密为 enc:v1: 密文。
func Seal(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	gcm, err := newGCM()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("secretbox: nonce: %w", err)
	}
	ct := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return prefix + base64.StdEncoding.EncodeToString(ct), nil
}

// Open 解密;非 enc:v1: 前缀的值按明文原样返回(历史/明文兼容)。
func Open(s string) (string, error) {
	if !strings.HasPrefix(s, prefix) {
		return s, nil
	}
	gcm, err := newGCM()
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(s, prefix))
	if err != nil || len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("secretbox: bad ciphertext")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("secretbox: decrypt: %w", err)
	}
	return string(plain), nil
}

func newGCM() (cipher.AEAD, error) {
	block, err := aes.NewCipher(keyFn())
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
