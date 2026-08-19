package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeAlarm 桩 device.AlarmService。
type fakeAlarm struct {
	taskNo string
	scope  string
}

func (f *fakeAlarm) ListAlarms(context.Context, int64) ([]device.Alarm, error) { return nil, nil }
func (f *fakeAlarm) CreateAlarm(context.Context, device.Alarm) (int64, error)  { return 0, nil }
func (f *fakeAlarm) UpdateAlarmStatus(context.Context, int64, string) error    { return nil }
func (f *fakeAlarm) AppendRetestTask(_ context.Context, scope string) (string, error) {
	f.scope = scope
	return f.taskNo, nil
}

// fakeDevice 桩 device.DeviceService(嵌入接口零实现)。
type fakeDevice struct{ device.DeviceService }

// TestBatchRetest 契约:批量复测按片区受理,返回任务号与范围。
func TestBatchRetest(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	al := &fakeAlarm{taskNo: "RT-20250817-0042"}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{
		User: &fakeUser{permOk: true}, Alarm: al, Device: &fakeDevice{},
	}, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/alarms/batch-retest", `{"scope":"马尼拉东区"}`, authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			TaskNo string `json:"taskNo"`
			Scope  string `json:"scope"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 || body.Data.TaskNo != "RT-20250817-0042" || body.Data.Scope != "马尼拉东区" {
		t.Fatalf("body=%s", w.Body.String())
	}
	if al.scope != "马尼拉东区" {
		t.Fatalf("scope=%q", al.scope)
	}
}

// TestLogout 契约:退出登录返回 ok(token 无状态)。
func TestLogout(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{User: &fakeUser{permOk: true}}, mgr)

	w := postAuth(t, r, "/api/admin/v1/auth/logout", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			OK bool `json:"ok"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 || !body.Data.OK {
		t.Fatalf("body=%s", w.Body.String())
	}
}
