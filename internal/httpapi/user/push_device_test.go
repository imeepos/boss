package userapi

// 契约:POST /push/device(用户端,JWT 客户鉴权);RegistrationID 上报落设备注册表。

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/portal"
	pushdomain "github.com/ymm-001/boss/internal/domain/push"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func newPushDeviceRouter() (*gin.Engine, *auth.Manager, pushdomain.DevicesService) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("test-secret", time.Hour)
	devices := pushdomain.NewDevicesMemoryStore()
	r := gin.New()
	Register(r, &app.Application{Portal: portal.NewMemory(), PushDevices: devices}, mgr)
	return r, mgr, devices
}

func TestPortal_PushDeviceRegister(t *testing.T) {
	r, mgr, devices := newPushDeviceRouter()
	tok, _ := signCustomerToken(mgr, 7, "13800000001")

	w := userPortalDo(r, http.MethodPost, "/api/user/v1/push/device",
		`{"registrationId":"1507bfd3f9ac1e045a","vendor":"jpush"}`, tok)
	code, data := userPortalCode(t, w)
	if code != int(apitypes.CodeOK) || data["ok"] != true {
		t.Fatalf("register code=%d data=%v body=%s", code, data, w.Body.String())
	}
	ids, err := devices.RegistrationIDs(context.Background(), pushdomain.SubjectUser, 7)
	if err != nil || len(ids) != 1 {
		t.Fatalf("devices=%v err=%v", ids, err)
	}
}

func TestPortal_PushDeviceInvalid(t *testing.T) {
	r, mgr, _ := newPushDeviceRouter()
	tok, _ := signCustomerToken(mgr, 7, "13800000001")

	w := userPortalDo(r, http.MethodPost, "/api/user/v1/push/device",
		`{"registrationId":"short"}`, tok)
	code, _ := userPortalCode(t, w)
	if code != int(apitypes.CodeInvalidParam) {
		t.Fatalf("invalid registrationId should %d, got %d body=%s", apitypes.CodeInvalidParam, code, w.Body.String())
	}

	// 未鉴权 401。
	w2 := userPortalDo(r, http.MethodPost, "/api/user/v1/push/device",
		`{"registrationId":"1507bfd3f9ac1e045a"}`, "")
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated should 401, got %d", w2.Code)
	}
}
