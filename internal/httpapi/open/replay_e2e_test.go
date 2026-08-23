// 回放工具端到端 + fixture 与沙箱样例同步守护(M5)。
package openapi

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"

	"github.com/ymm-001/boss/internal/domain/openplat"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

func repoPath(elems ...string) string {
	_, thisFile, _, _ := runtime.Caller(0)
	base := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	return filepath.Join(append([]string{base}, elems...)...)
}

// newSandboxRouter 真实开放面路由 + 沙箱应用桩(op_sb/ops_sb)。
func newSandboxRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	limiter := middleware.NewOpenRateLimiter()
	v1 := r.Group("/api/open/v1")
	v1.Use(middleware.OpenAuth(selftestFake{}), limiter.OpenRateLimit())
	registerOpenRoutes(v1, &app.Application{})
	return r
}

// TestReplayFixturesMatchSandboxSamples 守护:fixture 里的样例订单数据必须与
// sandbox.go 编译期样例一致(防三处同源漂移:Go 样例 / fixture / selftest 脚本)。
func TestReplayFixturesMatchSandboxSamples(t *testing.T) {
	b, err := os.ReadFile(repoPath("scripts", "openplat", "fixtures", "sandbox-replay.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Cases []struct {
			Name    string `json:"name"`
			Request struct {
				Path string `json:"path"`
			} `json:"request"`
			Expect struct {
				Data *struct {
					OrderNo string `json:"orderNo"`
					Status  string `json:"status"`
					Stage   int8   `json:"stage"`
				} `json:"data"`
			} `json:"expect"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &fx); err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, tc := range fx.Cases {
		d := tc.Expect.Data
		if d == nil {
			continue
		}
		raw, ok := openplat.SandboxOrderByNo(d.OrderNo)
		if !ok {
			t.Fatalf("case %s: sample %s not in sandbox set", tc.Name, d.OrderNo)
		}
		var got struct {
			OrderNo string `json:"orderNo"`
			Status  string `json:"status"`
			Stage   int8   `json:"stage"`
		}
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if got != *d {
			t.Fatalf("case %s: fixture %+v != sandbox sample %+v", tc.Name, d, got)
		}
		checked++
	}
	if checked < 2 {
		t.Fatalf("fixture should cover at least 2 sample orders, got %d", checked)
	}
}

// TestReplayToolAgainstOpenRouter 真实路由回放 fixture,期望全部 PASS。
func TestReplayToolAgainstOpenRouter(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	srv := httptest.NewServer(newSandboxRouter())
	defer srv.Close()

	cmd := exec.Command(node, repoPath("scripts", "openplat-replay.mjs"),
		"--base", srv.URL, "--app", "op_sb", "--secret", "ops_sb")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("replay failed: %v\n%s", err, out)
	}
	if want := "6/6 项通过"; !contains(string(out), want) {
		t.Fatalf("want %q in output:\n%s", want, out)
	}
}
