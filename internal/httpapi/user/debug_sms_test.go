package userapi

// 开发模式 debug SMS 端点测试:env 开关 + 公开/鉴权两条路由 + 无记录 40400 + 错 scene 42200。
//
// 注意:debugSmsEnabled 是包级变量,初始由 os.Getenv("BOSS_DEBUG_SMS") 读取;
// 这里每个用例临时改写该变量,函数退出时通过 t.Cleanup 复原。

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/pkg/auth"
)

func setDebugSms(t *testing.T, enabled bool) {
	t.Helper()
	prev := debugSmsEnabled
	debugSmsEnabled = enabled
	t.Cleanup(func() { debugSmsEnabled = prev })
}

func TestDebugSms_Disabled(t *testing.T) {
	setDebugSms(t, false)
	r, _, _ := newUserPortalRouter(userPortalCust(), nil, nil)
	w := userPortalDo(r, http.MethodGet,
		"/api/user/v1/debug/sms-code?phone=13800001234&scene=login", "", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("disable 时路由应 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestDebugSms_PublicHappyPath(t *testing.T) {
	setDebugSms(t, true)
	r, _, _ := newUserPortalRouter(userPortalCust(), nil, nil)
	// 先发码(走公开 /auth/sms-code),再回显
	if w := userPortalDo(r, http.MethodPost, "/api/user/v1/auth/sms-code",
		`{"phone":"13800001234","scene":"login"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("issue sms: %d %s", w.Code, w.Body.String())
	}
	w := userPortalDo(r, http.MethodGet,
		"/api/user/v1/debug/sms-code?"+url.Values{"phone": {"13800001234"}, "scene": {"login"}}.Encode(), "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("query: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":"123456"`) {
		t.Fatalf("code 未回显: %s", w.Body.String())
	}
}

func TestDebugSms_NoRecord404(t *testing.T) {
	setDebugSms(t, true)
	r, _, _ := newUserPortalRouter(userPortalCust(), nil, nil)
	w := userPortalDo(r, http.MethodGet,
		"/api/user/v1/debug/sms-code?"+url.Values{"phone": {"13900009999"}, "scene": {"login"}}.Encode(), "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("envelope: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":40400`) {
		t.Fatalf("无记录应 40400: %s", w.Body.String())
	}
}

func TestDebugSms_InvalidScene422(t *testing.T) {
	setDebugSms(t, true)
	r, _, _ := newUserPortalRouter(userPortalCust(), nil, nil)
	w := userPortalDo(r, http.MethodGet,
		"/api/user/v1/debug/sms-code?"+url.Values{"phone": {"13800001234"}, "scene": {"unknown"}}.Encode(), "", "")
	if !strings.Contains(w.Body.String(), `"code":42200`) {
		t.Fatalf("非法 scene 应 42200: %s", w.Body.String())
	}
}

func TestDebugSms_VerifyFromJWT(t *testing.T) {
	setDebugSms(t, true)
	cust := userPortalCust()
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("test-secret", time.Hour)
	tok, _ := signCustomerToken(mgr, 7, cust.Phone)
	r, _, _ := newUserPortalRouter(cust, nil, nil)

	// 走 uauth verify 端点发码(需登录),无需 phone 参数
	if w := userPortalDo(r, http.MethodPost, "/api/user/v1/auth/verify/sms-code", "", tok); w.Code != http.StatusOK {
		t.Fatalf("verify issue: %d %s", w.Code, w.Body.String())
	}
	w := userPortalDo(r, http.MethodGet,
		"/api/user/v1/debug/sms-code?scene=verify", "", tok)
	if w.Code != http.StatusOK {
		t.Fatalf("verify query: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":"123456"`) {
		t.Fatalf("verify scene 未从 JWT 解 phone 回显: %s", w.Body.String())
	}
}