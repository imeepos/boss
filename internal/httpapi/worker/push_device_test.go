package workerapi

// 契约:POST /push/device(师傅端,workerJWT 鉴权);RegistrationID 上报落设备注册表。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/portal"
	pushdomain "github.com/ymm-001/boss/internal/domain/push"
)

func newWorkerPushDeviceRouter(t *testing.T) (*gin.Engine, pushdomain.DevicesService) {
	t.Helper()
	t.Setenv("BOSS_JWT_SECRET", "portal-test-secret")
	gin.SetMode(gin.TestMode)
	devices := pushdomain.NewDevicesMemoryStore()
	r := gin.New()
	Register(r, &app.Application{Portal: portal.NewMemory(), PushDevices: devices}, newWorkerJWTManager())
	return r, devices
}

func workerPushDo(r *gin.Engine, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/worker/v1/push/device", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestWorkerPushDeviceRegister(t *testing.T) {
	r, devices := newWorkerPushDeviceRouter(t)
	tok, err := signWorkerToken(7, "张师傅")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	w := workerPushDo(r, `{"registrationId":"1507bfd3f9ac1e045a","vendor":"jpush"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Fatalf("register code=%d body=%s", resp.Code, w.Body.String())
	}
	ids, err := devices.RegistrationIDs(context.Background(), pushdomain.SubjectWorker, 7)
	if err != nil || len(ids) != 1 {
		t.Fatalf("devices=%v err=%v", ids, err)
	}
}

func TestWorkerPushDeviceGuard(t *testing.T) {
	r, _ := newWorkerPushDeviceRouter(t)

	// 未鉴权 → 401(中间件层)。
	w := workerPushDo(r, `{"registrationId":"1507bfd3f9ac1e045a"}`, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated should 401, got %d", w.Code)
	}
	// admin token(issuer=boss)不得通过师傅端校验。
	w2 := workerPushDo(r, `{"registrationId":"1507bfd3f9ac1e045a"}`, mustAdminToken(t))
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("admin token should 401 on worker portal, got %d", w2.Code)
	}
}
