// 自助验收工具端到端测试:真实 gin 路由 + node scripts/openplat-selftest.mjs。
// 无 node 环境时跳过(102 部署机与本地均有 node>=22)。
package openapi

import (
	"context"
	"errors"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/openplat"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

// selftestFake 最小 openplat.Service 桩:单沙箱应用 op_sb / secret ops_sb。
type selftestFake struct{}

func (selftestFake) CreateApp(context.Context, string, int, int, bool, int64) (*openplat.AppCreateResult, error) {
	return nil, errors.New("not implemented")
}
func (selftestFake) ListApps(context.Context) ([]openplat.App, error) { return nil, nil }
func (selftestFake) SetAppStatus(context.Context, int64, int16) error { return nil }
func (selftestFake) CreateSubscription(context.Context, int64, string, string) (*openplat.Subscription, error) {
	return nil, errors.New("not implemented")
}
func (selftestFake) ListSubscriptions(context.Context, int64) ([]openplat.Subscription, error) {
	return nil, nil
}
func (selftestFake) DeleteSubscription(context.Context, int64) error { return nil }
func (selftestFake) ListDeliveries(context.Context, int64) ([]openplat.Delivery, error) {
	return nil, nil
}
func (selftestFake) Requeue(context.Context, int64) error { return nil }
func (selftestFake) TouchUsage(context.Context, int64, time.Time) (int64, error) {
	return 1, nil
}
func (selftestFake) LookupActive(_ context.Context, appID string) (*openplat.AuthContext, error) {
	if appID != "op_sb" {
		return nil, openplat.ErrNotFound
	}
	return &openplat.AuthContext{AppRowID: 1, Secret: "ops_sb", RateLimitRPM: 1000, DailyQuota: 10000, Sandbox: true}, nil
}

// TestSelftestScriptAgainstOpenRouter 用真实路由跑集成方自助验收脚本,期望全部 PASS。
func TestSelftestScriptAgainstOpenRouter(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// 复用 Register 的鉴权链,但 Service 换桩:直接组装同构路由组。
	limiter := middleware.NewOpenRateLimiter()
	v1 := r.Group("/api/open/v1")
	v1.Use(middleware.OpenAuth(selftestFake{}), limiter.OpenRateLimit())
	registerOpenRoutes(v1, &app.Application{})

	srv := httptest.NewServer(r)
	defer srv.Close()

	_, thisFile, _, _ := runtime.Caller(0)
	script := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "scripts", "openplat-selftest.mjs")

	cmd := exec.Command(node, script, "--base", srv.URL, "--app", "op_sb", "--secret", "ops_sb")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("selftest failed: %v\n%s", err, out)
	}
	if want := "5/5 项通过"; !contains(string(out), want) {
		t.Fatalf("want %q in output:\n%s", want, out)
	}
}

// TestSelftestScriptRejectsBadSecret 错误 Secret 必须验收失败(反向断言)。
func TestSelftestScriptRejectsBadSecret(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	limiter := middleware.NewOpenRateLimiter()
	v1 := r.Group("/api/open/v1")
	v1.Use(middleware.OpenAuth(selftestFake{}), limiter.OpenRateLimit())
	registerOpenRoutes(v1, &app.Application{})

	srv := httptest.NewServer(r)
	defer srv.Close()

	_, thisFile, _, _ := runtime.Caller(0)
	script := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "scripts", "openplat-selftest.mjs")

	cmd := exec.Command(node, script, "--base", srv.URL, "--app", "op_sb", "--secret", "WRONG")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("bad secret must fail selftest:\n%s", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
