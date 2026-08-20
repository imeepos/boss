package workerapi

// W 师傅端 API key 鉴权测试:worker 主体密钥注入师傅身份,
// account/customer 主体密钥一律 401(与后台/客户身份不混用)。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/portal"
)

// fakeAPIKeySvc 内存版 apikey.Service:明文 key → 主体(哈希口径与 middleware 一致)。
type fakeAPIKeySvc struct {
	apikey.Service
	key   string
	subj  apikey.Subject
	found bool
}

func (f *fakeAPIKeySvc) Lookup(_ context.Context, keyHash string) (*apikey.Subject, error) {
	sum := sha256.Sum256([]byte(f.key))
	if f.found && keyHash == hex.EncodeToString(sum[:]) {
		s := f.subj
		return &s, nil
	}
	return nil, apikey.ErrNotFound
}

func (f *fakeAPIKeySvc) Touch(context.Context, string) {}

// apikeyTestRouter 装配带 API key service 的测试路由(worker 表内有 ID=7 张师傅)。
func apikeyTestRouter(t *testing.T, keys apikey.Service) *gin.Engine {
	t.Helper()
	t.Setenv("BOSS_JWT_SECRET", "portal-test-secret")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	a := &app.Application{
		Worker: &fakePortalWorkerSvc{},
		WorkOrder: &fakePortalWorkOrder{tickets: []order.DispatchTicket{
			{TicketNo: "TK-1", Status: "PENDING"},
		}},
		Order:  &fakePortalOrder{},
		Portal: portal.NewMemory(),
		APIKey: keys,
	}
	Register(r, a, newWorkerJWTManager())
	return r
}

func apikeyDo(r *gin.Engine, path, key string) map[string]any {
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("X-API-Key", key)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out == nil {
		out = map[string]any{}
	}
	out["_status"] = w.Code
	return out
}

func TestWorkerAPIKeyAuth(t *testing.T) {
	cases := []struct {
		name    string
		subj    apikey.Subject
		want    int
		wantMsg string
	}{
		{"worker 主体放行", apikey.Subject{Type: apikey.SubjectWorker, Ref: 7}, 200, ""},
		{"account 主体拒绝", apikey.Subject{Type: apikey.SubjectAccount, Ref: 103}, 401, "api key subject not found"},
		{"customer 主体拒绝", apikey.Subject{Type: apikey.SubjectCustomer, Ref: 213}, 401, "api key subject not found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			keys := &fakeAPIKeySvc{key: "h1", subj: tc.subj, found: true}
			out := apikeyDo(apikeyTestRouter(t, keys), "/api/worker/v1/tickets", "h1")
			if statusOf(out) != tc.want {
				t.Fatalf("status=%v resp=%v", out["_status"], out)
			}
			if tc.wantMsg != "" {
				msg, _ := out["msg"].(string)
				if !strings.Contains(msg, tc.wantMsg) {
					t.Fatalf("msg=%q want contains %q", msg, tc.wantMsg)
				}
			}
		})
	}
}

func TestWorkerAPIKeyInvalidKeyRejected(t *testing.T) {
	out := apikeyDo(apikeyTestRouter(t, &fakeAPIKeySvc{found: false}), "/api/worker/v1/tickets", "bad")
	if statusOf(out) != 401 {
		t.Fatalf("status=%v resp=%v", out["_status"], out)
	}
}

func TestWorkerJWTStillWorks(t *testing.T) {
	t.Setenv("BOSS_JWT_SECRET", "portal-test-secret")
	tok, err := signWorkerToken(7, "张师傅")
	if err != nil {
		t.Fatal(err)
	}
	out := portalWorkerDo(apikeyTestRouter(t, &fakeAPIKeySvc{found: false}), "GET", "/api/worker/v1/tickets", "", tok)
	if statusOf(out) != 200 {
		t.Fatalf("status=%v resp=%v", out["_status"], out)
	}
}

// statusOf 取测试响应状态码(json 断言与直接 set 的 int 兼容)。
func statusOf(out map[string]any) int {
	switch v := out["_status"].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return -1
}
