package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeProvision 桩 provision.ProvisionService。
type fakeProvision struct {
	task    *provision.Task
	logs    []provision.Log
	retried *struct {
		id      int64
		retries int16
	}
}

func (f *fakeProvision) ListTemplates(context.Context) ([]provision.Template, error) { return nil, nil }
func (f *fakeProvision) CreateTemplate(context.Context, provision.Template) (int64, error) {
	return 0, nil
}
func (f *fakeProvision) ListTasks(context.Context) ([]provision.Task, error) { return nil, nil }
func (f *fakeProvision) CreateTask(context.Context, provision.Task) (int64, error) {
	return 0, nil
}
func (f *fakeProvision) GetTaskByNo(context.Context, string) (*provision.Task, error) {
	return f.task, nil
}
func (f *fakeProvision) ListLogs(context.Context, int64) ([]provision.Log, error) {
	return f.logs, nil
}
func (f *fakeProvision) AppendLog(context.Context, provision.Log) (int64, error) { return 0, nil }
func (f *fakeProvision) ExecuteTask(context.Context, int64) error                { return nil }
func (f *fakeProvision) FailTask(context.Context, int64, string) error           { return nil }
func (f *fakeProvision) RetryTask(_ context.Context, id int64, retries int16) error {
	f.retried = &struct {
		id      int64
		retries int16
	}{id, retries}
	return nil
}

// TestRetryProvisionTask 契约:失败任务按 taskNo 重试,重试计数取日志最大值+1。
func TestRetryProvisionTask(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeProvision{
		task: &provision.Task{ID: 3, TaskNo: "PT-3", Status: "FAILED"},
		logs: []provision.Log{{ID: 1, TaskID: 3, Result: "RETRY", Retries: 2}},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{User: &fakeUser{permOk: true}, Provision: f}, mgr)

	w := postAuth(t, r, "/api/v1/provision-tasks/PT-3/retry", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 {
		t.Fatalf("body=%s", w.Body.String())
	}
	if f.retried == nil || f.retried.id != 3 || f.retried.retries != 2 {
		t.Fatalf("retried=%+v", f.retried)
	}
}
