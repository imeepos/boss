package workerapi

// 入驻场景发码契约(2026-08-25 修复:入驻页发码此前被 login 场景的在职校验死锁):
// scene=register 对新手机号公开;scene=login(默认)仍须在职师傅;两场景码分储不串用。

import (
	"testing"
)

func TestSmsCodeRegisterScene(t *testing.T) {
	t.Setenv("BOSS_DEV_MODE", "true") // 注册 /auth/dev/sms-code 回显端点
	r := portalTestRouter(t, &fakePortalWorkOrder{}, &fakePortalOrder{})

	// register 场景:陌生手机号(非在职师傅)发码成功
	if res := portalWorkerDo(r, "POST", "/api/worker/v1/auth/sms-code",
		`{"phone":"13900001111","scene":"register"}`, ""); res["code"].(float64) != 0 {
		t.Fatalf("register scene should be public: %v", res)
	}
	// dev 回显 register 场景拿到码(内存替身固定 123456)
	res := portalWorkerDo(r, "POST", "/api/worker/v1/auth/dev/sms-code",
		`{"phone":"13900001111","scene":"register"}`, "")
	if res["code"].(float64) != 0 {
		t.Fatalf("dev echo register failed: %v", res)
	}
	if got := res["data"].(map[string]any)["code"].(string); got != "123456" {
		t.Fatalf("register scene code = %q, want 123456", got)
	}
	// login 场景(默认):同一手机号仍被在职校验拦截(40100)
	if res := portalWorkerDo(r, "POST", "/api/worker/v1/auth/sms-code",
		`{"phone":"13900001111"}`, ""); res["code"].(float64) != 40100 {
		t.Fatalf("login scene should require employed worker: %v", res)
	}
	// 非法 scene 拒绝
	if res := portalWorkerDo(r, "POST", "/api/worker/v1/auth/sms-code",
		`{"phone":"13900001111","scene":"xx"}`, ""); res["code"].(float64) != 42200 {
		t.Fatalf("invalid scene should be 42200: %v", res)
	}
	// dev 回显 login 场景查不到 register 码(分储不串用)→ 40400
	if res := portalWorkerDo(r, "POST", "/api/worker/v1/auth/dev/sms-code",
		`{"phone":"13900001111","scene":"login"}`, ""); res["code"].(float64) != 40400 {
		t.Fatalf("login echo should not see register code: %v", res)
	}
}
