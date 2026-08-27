package license

import (
	"encoding/json"
	"os"
	"testing"
)

// TestReal102Token 用 release-platform(102)真实签发的离线令牌验证验签内核。
// 令牌文件由 scripts/issue-offline-token.sh 或手工 curl 生成;文件缺失时跳过
// (本地开发无 102 凭据也不打断门禁),102 CI 环境放置后自动生效。
func TestReal102Token(t *testing.T) {
	data, err := os.ReadFile("/tmp/offline-token.json")
	if err != nil {
		t.Skipf("real 102 token not present: %v", err)
	}
	var resp struct {
		Token struct {
			Payload   string `json:"payload"`
			KeyID     string `json:"key_id"`
			Signature string `json:"signature"`
			Algorithm string `json:"algorithm"`
		} `json:"token"`
		PublicKeyHex string `json:"public_key_hex"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatal(err)
	}
	tok := Token{
		Payload:   resp.Token.Payload,
		KeyID:     resp.Token.KeyID,
		Signature: resp.Token.Signature,
		Algorithm: resp.Token.Algorithm,
	}
	raw, _ := json.Marshal(tok)
	v, err := NewVerifier(resp.PublicKeyHex)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := v.Verify(raw, VerifyOptions{
		ExpectedDeviceID:    "boss-license-dev-01",
		ExpectedFingerprint: "fp-boss-01",
	})
	if err != nil {
		t.Fatalf("real 102 token verify FAILED: %v", err)
	}
	t.Logf("real 102 token OK: license=%s product=%s type=%s", claims.LicenseID, claims.ProductID, claims.LicenseType)
}