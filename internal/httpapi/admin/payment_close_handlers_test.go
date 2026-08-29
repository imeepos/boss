package adminapi

// 柜台日结/柜面收款风控 handler 测试:写侧本人强约束、渠道分流映射。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// tokenAs 按账号名/角色签管理端 JWT。
func tokenAs(t *testing.T, mgr *auth.Manager, username, role string) string {
	t.Helper()
	tok, err := mgr.Sign(auth.AudAdmin, 9, username, role)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// newCloseRouter 柜台日结路由测试器:fakeBilling 恒平账。
func newCloseRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{
		User:    &fakeUser{permOk: true},
		Billing: &fakeBilling{},
	}, auth.NewManager("s", time.Hour))
	return r
}

// 契约(纪要 2026-08-28 待定项收口 Item2):非 sysadmin 只能回填本人名下日结,
// 他人行 42200;sysadmin 可代填;operatorName 缺省落本人。
func TestSaveDailyClosing_SelfConstraint(t *testing.T) {
	r := newCloseRouter()
	mgr := auth.NewManager("s", time.Hour)
	cases := []struct {
		name, token, operator string
		wantCode              int // envelope code;0=OK
	}{
		{"柜员回填他人行被拒", tokenAs(t, mgr, "cashier_wang", "ops"), "admin", 42200},
		{"柜员回填本人行放行", tokenAs(t, mgr, "cashier_wang", "ops"), "cashier_wang", 0},
		{"operatorName 缺省落本人", tokenAs(t, mgr, "cashier_wang", "ops"), "", 0},
		{"sysadmin 可代填", tokenAs(t, mgr, "boss", "sysadmin"), "admin", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"date":"2026-08-28","siteName":"旗舰店","operatorName":"` + tc.operator + `","countedAmount":100}`
			req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/daily-closings", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tc.token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			resp := w.Body.String()
			if tc.wantCode == 0 && !strings.Contains(resp, `"code":0`) {
				t.Fatalf("want ok, got %s", resp)
			}
			if tc.wantCode != 0 && !strings.Contains(resp, "本人") {
				t.Fatalf("want self-constraint reject, got %s", resp)
			}
		})
	}
}

// 渠道→资金通道映射测试在 internal/domain/billing(payment_guard_test.go TestChannelMethods)。
