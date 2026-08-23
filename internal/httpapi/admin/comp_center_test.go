package adminapi

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/report"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

type fakeCompTaskStore struct {
	tasks  map[int64]*report.CompTask
	nextID int64
}

func newFakeCompTaskStore() *fakeCompTaskStore {
	return &fakeCompTaskStore{tasks: map[int64]*report.CompTask{}, nextID: 1}
}
func (f *fakeCompTaskStore) ListCompTasks(_ context.Context, q report.CompTaskFilter) ([]report.CompTask, int, error) {
	out := []report.CompTask{}
	for _, t := range f.tasks {
		if q.Status != "" && t.Status != q.Status {
			continue
		}
		out = append(out, *t)
	}
	return out, len(out), nil
}
func (f *fakeCompTaskStore) GetCompTask(_ context.Context, id int64) (*report.CompTask, error) {
	t, ok := f.tasks[id]
	if !ok {
		return nil, report.ErrCompTaskNotFound
	}
	return t, nil
}
func (f *fakeCompTaskStore) CreateCompTask(_ context.Context, t *report.CompTask) error {
	t.ID = f.nextID
	f.nextID++
	f.tasks[t.ID] = t
	return nil
}
func (f *fakeCompTaskStore) UpdateCompTask(_ context.Context, t *report.CompTask) error {
	f.tasks[t.ID] = t
	return nil
}
func (f *fakeCompTaskStore) ClaimCompTask(_ context.Context, id, actor int64, name string) error {
	t, ok := f.tasks[id]
	if !ok {
		return report.ErrCompTaskNotFound
	}
	if t.Status != report.TaskStatusOpen {
		return report.ErrIllegalStatus
	}
	t.Status = report.TaskStatusClaimed
	return nil
}
func (f *fakeCompTaskStore) TransferCompTask(_ context.Context, id, from, to int64, name, actor string) error {
	t, ok := f.tasks[id]
	if !ok {
		return report.ErrCompTaskNotFound
	}
	t.AssigneeID = &to
	t.AssigneeName = name
	return nil
}
func (f *fakeCompTaskStore) CloseCompTask(_ context.Context, id, actor int64, name, reason string) error {
	t, ok := f.tasks[id]
	if !ok {
		return report.ErrCompTaskNotFound
	}
	if t.Status == report.TaskStatusClosed {
		return report.ErrIllegalStatus
	}
	t.Status = report.TaskStatusClosed
	t.CloseReason = reason
	return nil
}
func (f *fakeCompTaskStore) IncrementRetry(_ context.Context, id int64) error {
	t, ok := f.tasks[id]
	if !ok {
		return report.ErrCompTaskNotFound
	}
	t.RetryCount++
	return nil
}
func (f *fakeCompTaskStore) BatchInsertCompTasks(_ context.Context, ts []report.CompTask) error {
	for i := range ts {
		_ = f.CreateCompTask(context.Background(), &ts[i])
	}
	return nil
}

type fakePatrolStore struct{}

func (*fakePatrolStore) UpsertSnapshot(context.Context, *report.Snapshot) error { return nil }
func (*fakePatrolStore) LatestSnapshot(context.Context, string) (*report.Snapshot, error) {
	return nil, report.ErrNoSnapshot
}
func (*fakePatrolStore) LatestSnapshots(context.Context, string, int) ([]report.Snapshot, error) {
	return nil, nil
}
func (*fakePatrolStore) ListSnapshots(context.Context) ([]report.Snapshot, error) { return nil, nil }
func (*fakePatrolStore) SnapshotByID(context.Context, int64) (*report.Snapshot, error) {
	return nil, report.ErrNoSnapshot
}

func TestCompTaskRoutes(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	st := newFakeCompTaskStore()
	_ = st.CreateCompTask(context.Background(), &report.CompTask{Source: "order", BizType: "order", BizID: "ORD-001", Priority: report.PriorityHigh, Status: report.TaskStatusOpen})
	_ = st.CreateCompTask(context.Background(), &report.CompTask{Source: "billing", BizType: "payment", BizID: "PAY-001", Priority: report.PriorityNormal, Status: report.TaskStatusOpen})
	router := func() *gin.Engine {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		Register(r, &app.Application{User: &fakeUser{permOk: true}, CompTask: report.NewCompTaskService(st), Report: &report.ReportService{St: &fakePatrolStore{}}}, mgr)
		return r
	}
	w := getJSON(t, router(), "/api/admin/v1/comp-tasks", token)
	if w.Code != 200 || !contains(w.Body.String(), "ORD-001") {
		t.Fatalf("list: %d %s", w.Code, w.Body)
	}
	w = getJSON(t, router(), "/api/admin/v1/comp-tasks/1", token)
	if w.Code != 200 {
		t.Fatalf("get: %d", w.Code)
	}
	w = postAuth(t, router(), "/api/admin/v1/comp-tasks/1/claim", token)
	if envCode(t, w) != apitypes.CodeOK {
		t.Fatalf("claim: %s", w.Body)
	}
	w = postAuth(t, router(), "/api/admin/v1/comp-tasks/1/claim", token)
	if envCode(t, w) != apitypes.CodeStateInvalid {
		t.Fatalf("claim twice: %s", w.Body)
	}
	w = postJSONAuth(t, router(), "/api/admin/v1/comp-tasks/2/transfer", `{"toId":2,"toName":"tech-b"}`, token)
	if envCode(t, w) != apitypes.CodeOK {
		t.Fatalf("transfer: %s", w.Body)
	}
	w = postAuth(t, router(), "/api/admin/v1/comp-tasks/2/retry", token)
	if envCode(t, w) != apitypes.CodeOK {
		t.Fatalf("retry: %s", w.Body)
	}
	w = postJSONAuth(t, router(), "/api/admin/v1/comp-tasks/2/close", `{"reason":"handled"}`, token)
	if envCode(t, w) != apitypes.CodeOK {
		t.Fatalf("close: %s", w.Body)
	}
	w = postAuth(t, router(), "/api/admin/v1/comp-tasks/2/close", token)
	if envCode(t, w) != apitypes.CodeStateInvalid {
		t.Fatalf("close twice: %s", w.Body)
	}
	w = postAuth(t, router(), "/api/admin/v1/comp-tasks/2/replay", token)
	if envCode(t, w) != apitypes.CodeOK {
		t.Fatalf("replay: %s", w.Body)
	}
}
