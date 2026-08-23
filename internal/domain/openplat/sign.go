package openplat

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

// CanonicalString 构造签名串:appId\nMETHOD\npath\ntimestamp\nnonce\nsha256hex(body)。
// path 为路由路径(不含 query);body 为空时其哈希仍参与串拼接。
func CanonicalString(appID, method, path, timestamp, nonce string, body []byte) string {
	bodyHash := sha256.Sum256(body)
	return appID + "\n" + method + "\n" + path + "\n" + timestamp + "\n" + nonce + "\n" + hex.EncodeToString(bodyHash[:])
}

// Sign 用应用 Secret 对签名串做 HMAC-SHA256,返回 hex。
func Sign(secret, canonical string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyRequest 校验请求签名与时间戳窗口。
// signatureHeader 为调用方提供的 hex 签名;返回 (app行ID, 错误)。
func VerifyRequest(auth *AuthContext, r *http.Request, body []byte,
	appID, timestamp, nonce, signature string) error {
	ts, err := parseUnix(timestamp)
	if err != nil {
		return fmt.Errorf("%w: bad timestamp %q", ErrBadSignature, timestamp)
	}
	if diff := time.Now().Unix() - ts; diff < -TimestampWindow || diff > TimestampWindow {
		return ErrStaleTimestamp
	}
	canonical := CanonicalString(appID, r.Method, r.URL.Path, timestamp, nonce, body)
	if !hmac.Equal([]byte(Sign(auth.Secret, canonical)), []byte(signature)) {
		return ErrBadSignature
	}
	return nil
}

// SignPayload 对 Webhook 负载签名(M2 投递器用;hex)。
func SignPayload(secret string, timestamp string, body []byte) string {
	bodyHash := sha256.Sum256(body)
	return Sign(secret, timestamp+"\n"+hex.EncodeToString(bodyHash[:]))
}

func parseUnix(s string) (int64, error) {
	var v int64
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil || v <= 0 {
		return 0, fmt.Errorf("invalid unix seconds %q", s)
	}
	return v, nil
}
