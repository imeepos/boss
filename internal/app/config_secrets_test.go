package app

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/ymm-001/boss/internal/pkg/secretbox"
)

// TestDecryptConfigSecretFailuresSurface 回归(密钥轮换):
// 密文解密失败不得静默当"未配置"——必须输出含字段名的告警日志并回退空串(env 兜底);
// 热重载场景同一字段只告警一次,不刷屏。
func TestDecryptConfigSecretFailuresSurface(t *testing.T) {
	valid, err := secretbox.Seal("real-secret")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if got := decryptConfigSecret("t.valid", valid); got != "real-secret" {
		t.Fatalf("valid roundtrip: %q", got)
	}
	if got := decryptConfigSecret("t.empty", ""); got != "" {
		t.Fatalf("empty passthrough: %q", got)
	}

	var buf strings.Builder
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	for i := 0; i < 3; i++ { // 模拟多轮热重载
		if got := decryptConfigSecret("t.rotated", "enc:v1:not-a-valid-ciphertext"); got != "" {
			t.Fatalf("rotated ciphertext should fall back to empty, got %q", got)
		}
	}
	out := buf.String()
	if n := strings.Count(out, "[config-secrets] DECRYPT FAILED"); n != 1 {
		t.Fatalf("warn count = %d, want exactly 1 per field:\n%s", n, out)
	}
	if !strings.Contains(out, "field=t.rotated") {
		t.Fatalf("warning must name the field:\n%s", out)
	}
}
