package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeWorkerOps 桩 worker 各服务口(仅接单设置/公告路由用到的方法落账)。
type fakeWorkerOps struct {
	w         *worker.Worker
	settings  *worker.Settings
	notices   []worker.Notice
	created   *worker.Notice
	toggled   int64
	reviewed  int64
	confirmed int64
	// 录入师傅/重置密码落账(worker_account_handlers 测试)。
	createdWorker   *worker.Worker
	createdPassword string
	pwdWorkerID     int64
	pwdPassword     string
}

func (f *fakeWorkerOps) ListGroups(context.Context) ([]worker.Group, error) { return nil, nil }
func (f *fakeWorkerOps) CreateGroup(context.Context, worker.Group) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerOps) ListWorkers(context.Context, int64, string) ([]worker.Worker, error) {
	return nil, nil
}
func (f *fakeWorkerOps) CreateWorker(context.Context, worker.Worker) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerOps) CreateWorkerWithPassword(_ context.Context, w worker.Worker, password string) (int64, error) {
	f.createdWorker = &w
	f.createdPassword = password
	return 501, nil
}
func (f *fakeWorkerOps) SetPassword(_ context.Context, workerID int64, password string) error {
	f.pwdWorkerID = workerID
	f.pwdPassword = password
	return nil
}
func (f *fakeWorkerOps) VerifyPassword(context.Context, int64, string) (bool, error) {
	return false, nil
}
func (f *fakeWorkerOps) GetWorker(context.Context, int64) (*worker.Worker, error) {
	return f.w, nil
}
func (f *fakeWorkerOps) ListMemberships(context.Context, int64) ([]worker.Membership, error) {
	return nil, nil
}
func (f *fakeWorkerOps) AppendMembership(context.Context, worker.Membership) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerOps) GetSettings(context.Context, int64) (*worker.Settings, error) {
	return f.settings, nil
}
func (f *fakeWorkerOps) UpsertSettings(_ context.Context, s worker.Settings) (int64, error) {
	f.settings = &s
	return 1, nil
}
func (f *fakeWorkerOps) ListMessages(context.Context, int64) ([]worker.Message, error) {
	return nil, nil
}
func (f *fakeWorkerOps) SendMessage(context.Context, worker.Message) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerOps) ListPerformances(context.Context, int64) ([]worker.Performance, error) {
	return nil, nil
}
func (f *fakeWorkerOps) UpsertPerformance(context.Context, worker.Performance) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerOps) ListCommissions(context.Context, int64) ([]worker.Commission, error) {
	return nil, nil
}
func (f *fakeWorkerOps) UpsertCommission(context.Context, worker.Commission) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerOps) ListSchedules(context.Context, int64) ([]worker.Schedule, error) {
	return nil, nil
}
func (f *fakeWorkerOps) UpsertSchedule(context.Context, worker.Schedule) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerOps) ListMaterials(context.Context, int64) ([]worker.Material, error) {
	return nil, nil
}
func (f *fakeWorkerOps) AppendMaterial(context.Context, worker.Material) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerOps) ListTools(context.Context, int64) ([]worker.Tool, error) {
	return nil, nil
}
func (f *fakeWorkerOps) AppendTool(context.Context, worker.Tool) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerOps) ListFeedbacks(context.Context, int64) ([]worker.Feedback, error) {
	return nil, nil
}
func (f *fakeWorkerOps) AppendFeedback(context.Context, worker.Feedback) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerOps) ListAssetReturns(context.Context, int64) ([]worker.AssetReturn, error) {
	return nil, nil
}
func (f *fakeWorkerOps) AppendAssetReturn(context.Context, worker.AssetReturn) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerOps) ReviewFeedback(_ context.Context, feedbackID int64) error {
	f.reviewed = feedbackID
	return nil
}
func (f *fakeWorkerOps) ConfirmAssetReturn(_ context.Context, returnID int64) error {
	f.confirmed = returnID
	return nil
}
func (f *fakeWorkerOps) ListNotices(context.Context) ([]worker.Notice, error) {
	return f.notices, nil
}
func (f *fakeWorkerOps) CreateNotice(_ context.Context, n worker.Notice) (int64, error) {
	f.created = &n
	return 11, nil
}
func (f *fakeWorkerOps) ToggleNotice(_ context.Context, id int64) error {
	f.toggled = id
	return nil
}

// putAuth 带鉴权令牌发起 PUT。
func putAuth(t *testing.T, r *gin.Engine, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// postBodyAuth 带鉴权令牌 + JSON body 发起 POST。
func postBodyAuth(t *testing.T, r *gin.Engine, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func newWorkerRouter(f *fakeWorkerOps, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{
		User: &fakeUser{permOk: true}, Worker: f, WorkerLedger: f, WorkerFact: f,
		WorkerEvent: f, WorkerNotice: f,
	}, mgr)
	return r
}

// TestSaveWorkerSettings 契约:修改接单设置(在线/半径/接单类型)即时落账。
func TestSaveWorkerSettings(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeWorkerOps{w: &worker.Worker{ID: 5, Name: "张师傅"}}
	r := newWorkerRouter(f, mgr)

	w := putAuth(t, r, "/api/admin/v1/workers/5/settings",
		`{"online":true,"radiusKm":30,"acceptTypes":["INSTALL","REPAIR"]}`, authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if f.settings == nil || !f.settings.Accepting || f.settings.RadiusKm != 30 ||
		f.settings.AcceptTypes != "INSTALL,REPAIR" || f.settings.WorkerID != 5 {
		t.Fatalf("settings=%+v", f.settings)
	}
}

// TestNoticeHandlers 契约:公告可查(含已下架)/发布/上下架切换。
func TestNoticeHandlers(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeWorkerOps{notices: []worker.Notice{
		{ID: 1, Title: "防水作业提示", Category: "安全作业提醒", Active: true},
	}}
	r := newWorkerRouter(f, mgr)

	w := getJSON(t, r, "/api/admin/v1/notices", authToken(t, mgr))
	var list struct {
		Code int `json:"code"`
		Data struct {
			Items []worker.Notice `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Code != 0 || len(list.Data.Items) != 1 || list.Data.Items[0].Title != "防水作业提示" {
		t.Fatalf("list=%+v", list)
	}

	w = postBodyAuth(t, r, "/api/admin/v1/notices", `{"title":"物料配发说明","category":"物料公告"}`, authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if f.created == nil || f.created.Title != "物料配发说明" || !f.created.Active {
		t.Fatalf("created=%+v", f.created)
	}

	w = putAuth(t, r, "/api/admin/v1/notices/1/toggle", "", authToken(t, mgr))
	if w.Code != http.StatusOK || f.toggled != 1 {
		t.Fatalf("status=%d toggled=%d", w.Code, f.toggled)
	}
}

// TestWorkerDetail 契约:师傅详情按 id 返回单条(worker.yaml GET /workers/{workerId})。
func TestWorkerDetail(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeWorkerOps{w: &worker.Worker{ID: 5, Name: "张师傅", StaffNo: "W-001"}}
	r := newWorkerRouter(f, mgr)

	w := getJSON(t, r, "/api/admin/v1/workers/5", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got struct {
		Code int           `json:"code"`
		Data worker.Worker `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Code != 0 || got.Data.ID != 5 || got.Data.Name != "张师傅" {
		t.Fatalf("got=%+v", got)
	}
}

// TestReviewFeedback 契约:差评复核即时落账(worker.yaml POST /worker-feedbacks/{feedbackId}/review)。
func TestReviewFeedback(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeWorkerOps{}
	r := newWorkerRouter(f, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/worker-feedbacks/3/review", "", authToken(t, mgr))
	if w.Code != http.StatusOK || f.reviewed != 3 {
		t.Fatalf("status=%d reviewed=%d", w.Code, f.reviewed)
	}
}

// TestConfirmAssetReturn 契约:确认返库即时落账(worker.yaml POST /asset-returns/{returnId}/confirm)。
func TestConfirmAssetReturn(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeWorkerOps{}
	r := newWorkerRouter(f, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/asset-returns/7/confirm", "", authToken(t, mgr))
	if w.Code != http.StatusOK || f.confirmed != 7 {
		t.Fatalf("status=%d confirmed=%d", w.Code, f.confirmed)
	}
}

func (f *fakeWorkerOps) AppendClock(context.Context, worker.Attendance) (int64, error) {
	return 1, nil
}

func (f *fakeWorkerOps) ListClocks(context.Context, int64, time.Time) ([]worker.Attendance, error) {
	return nil, nil
}

func (f *fakeWorkerOps) AppendSafetyCheck(context.Context, worker.SafetyCheck) (int64, error) {
	return 1, nil
}

func (f *fakeWorkerOps) ListSafetyChecks(context.Context, int64) ([]worker.SafetyCheck, error) {
	return nil, nil
}

func (f *fakeWorkerOps) GetMaterialItem(context.Context, int64) (*worker.MaterialItem, error) {
	return nil, nil
}
func (f *fakeWorkerOps) GetTool(context.Context, int64) (*worker.ToolItem, error) {
	return nil, nil
}
func (f *fakeWorkerOps) ListToolItems(context.Context) ([]worker.ToolItem, error) {
	return nil, nil
}
func (f *fakeWorkerOps) ListMaterialItems(context.Context) ([]worker.MaterialItem, error) {
	return nil, nil
}
func (f *fakeWorkerOps) AppendReplaceLog(context.Context, worker.ReplaceLog) (int64, error) {
	return 1, nil
}
func (f *fakeWorkerOps) ListReplaceLogs(context.Context, string) ([]worker.ReplaceLog, error) {
	return nil, nil
}
