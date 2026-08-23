package adminapi

// 契约:GET /compensation-tasks(补偿任务中心,menu:report 门禁)。

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/report"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeCompCenterStore 同时满足 report.Store 与 CompTaskLister 的最小桩。
type fakeCompCenterStore struct{ tasks []report.CompTaskView }

func (f *fakeCompCenterStore) UpsertSnapshot(context.Context, *report.Snapshot) error { return nil }
func (f *fakeCompCenterStore) LatestSnapshot(context.Context, string) (*report.Snapshot, error) {
	return nil, report.ErrNoSnapshot
}
func (f *fakeCompCenterStore) LatestSnapshots(context.Context, string, int) ([]report.Snapshot, error) {
	return nil, nil
}
func (f *fakeCompCenterStore) ListSnapshots(context.Context) ([]report.Snapshot, error) {
	return nil, nil
}
func (f *fakeCompCenterStore) SnapshotByID(context.Context, int64) (*report.Snapshot, error) {
	return nil, report.ErrNoSnapshot
}
func (f *fakeCompCenterStore) CompensationTasks(context.Context) ([]report.CompTaskView, error) {
	return f.tasks, nil
}

func TestCompensationTasksRoute(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	newRouter := func(permOk bool) *gin.Engine {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		Register(r, &app.Application{
			User: &fakeUser{permOk: permOk},
			Report: &report.ReportService{St: &fakeCompCenterStore{tasks: []report.CompTaskView{
				{Domain: "provision", Type: "provisionTask", RefID: "TASK-9", Status: "FAILED",
					RetryPath: "/api/admin/v1/provision-tasks/TASK-9/retry"},
			}}},
		}, mgr)
		return r
	}

	t.Run("无权限 403", func(t *testing.T) {
		w := getJSON(t, newRouter(false), "/api/admin/v1/compensation-tasks", token)
		if w.Code != 403 {
			t.Fatalf("status=%d want 403", w.Code)
		}
	})
	t.Run("清单含可回放路径", func(t *testing.T) {
		w := getJSON(t, newRouter(true), "/api/admin/v1/compensation-tasks", token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if !contains(w.Body.String(), "provision-tasks/TASK-9/retry") {
			t.Fatalf("body=%s", w.Body.String())
		}
	})
}
