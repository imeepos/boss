package app

// 缺口路由单测:faqs(复用 user_faqs)/device-maintenances/hall-items/
// service-messages/reports/账号停用,表驱动 happy path + 未命中分支。

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/analytics"
	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/domain/report"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

// fakeGapDevice 桩 device.DeviceService(嵌入接口零实现,仅覆写 ListMaintenances)。
type fakeGapDevice struct{ device.DeviceService }

func (fakeGapDevice) ListMaintenances(context.Context) ([]device.DeviceMaintenance, error) {
	return []device.DeviceMaintenance{{DeviceNo: "DEV-01"}}, nil
}

type fakeWorkerLedger struct{ sent worker.Message }

func (f *fakeWorkerLedger) ListMemberships(context.Context, int64) ([]worker.Membership, error) {
	return nil, nil
}
func (f *fakeWorkerLedger) AppendMembership(context.Context, worker.Membership) (int64, error) {
	return 1, nil
}
func (f *fakeWorkerLedger) GetSettings(context.Context, int64) (*worker.Settings, error) {
	return nil, nil
}
func (f *fakeWorkerLedger) UpsertSettings(context.Context, worker.Settings) (int64, error) {
	return 1, nil
}
func (f *fakeWorkerLedger) ListMessages(context.Context, int64) ([]worker.Message, error) {
	return []worker.Message{{ID: 1, Title: "调度会话"}}, nil
}
func (f *fakeWorkerLedger) SendMessage(_ context.Context, m worker.Message) (int64, error) {
	f.sent = m
	return 7, nil
}

type fakeAnalyticsGap struct{}

func (fakeAnalyticsGap) FiveIndicators(context.Context) ([]analytics.Indicator, []analytics.RegionROI, error) {
	return nil, nil, nil
}
func (fakeAnalyticsGap) Heatmap(context.Context) ([]analytics.HeatCell, error) {
	return nil, nil
}
func (fakeAnalyticsGap) MaintenanceList(context.Context) ([]analytics.MaintenanceItem, error) {
	return nil, nil
}

type fakeReportStore struct{ snap report.Snapshot }

func (f *fakeReportStore) UpsertSnapshot(_ context.Context, s *report.Snapshot) error {
	f.snap = *s
	if s.ID == 0 {
		f.snap.ID = 3
	}
	return nil
}
func (f *fakeReportStore) LatestSnapshot(context.Context, string) (*report.Snapshot, error) {
	return &f.snap, nil
}
func (f *fakeReportStore) ListSnapshots(context.Context) ([]report.Snapshot, error) {
	return nil, nil
}
func (f *fakeReportStore) SnapshotByID(context.Context, int64) (*report.Snapshot, error) {
	return &f.snap, nil
}

func newGapRouter(ud *fakeUserdata, fu *fakeUser, wl *fakeWorkerLedger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("s", time.Hour)
	r := gin.New()
	authed := r.Group("/api/v1")
	authed.Use(middleware.Authn(mgr))
	app := &Application{
		User: fu, UserData: ud, Device: fakeGapDevice{}, WorkerLedger: wl,
		Report: &report.ReportService{Ana: fakeAnalyticsGap{}, St: &fakeReportStore{}},
	}
	registerUserdataGapRoutes(authed, app)
	return r
}

func TestUserdataGapEndpoints_HappyPath(t *testing.T) {
	ud := &fakeUserdata{row: map[string]any{"faqId": "FAQ-01"}}
	fu := &fakeUser{permOk: true, accounts: []user.AccountRow{{
		ID: 5, Username: "ops5", RealName: "运营五", RoleCode: "ops", Status: 1,
	}}}
	wl := &fakeWorkerLedger{}
	r := newGapRouter(ud, fu, wl)
	tk := authToken(t, auth.NewManager("s", time.Hour))

	cases := []struct {
		method, path, body string
		wantCalled         string
	}{
		{"GET", "/api/v1/faqs", "", ""},
		{"POST", "/api/v1/faqs", `{"title":"如何缴费?","summary":"App内自助缴费"}`, ""},
		{"PUT", "/api/v1/faqs/FAQ-01/toggle", "", "FAQ-01"},
		{"GET", "/api/v1/device-maintenances", "", ""},
		{"GET", "/api/v1/hall-items", "", ""},
		{"GET", "/api/v1/service-messages?workerId=3", "", ""},
		{"POST", "/api/v1/service-messages", `{"workerId":3,"content":"请尽快上门"}`, ""},
		{"POST", "/api/v1/reports", `{"period":"weekly"}`, ""},
		{"DELETE", "/api/v1/accounts/5", "", ""},
	}
	for _, tc := range cases {
		w := doReq(t, r, tc.method, tc.path, tk, tc.body)
		assertOK(t, w)
		if tc.wantCalled != "" && ud.calledPath != tc.wantCalled {
			t.Fatalf("%s %s: called=%q want %q", tc.method, tc.path, ud.calledPath, tc.wantCalled)
		}
	}
	if wl.sent.WorkerID != 3 || wl.sent.Title != "调度会话" || wl.sent.Level != "INFO" {
		t.Fatalf("service message not defaulted: %+v", wl.sent)
	}
}

func TestUserdataGapAccountNotFound(t *testing.T) {
	ud := &fakeUserdata{row: map[string]any{}}
	fu := &fakeUser{permOk: true}
	r := newGapRouter(ud, fu, &fakeWorkerLedger{})
	mgr := auth.NewManager("s", time.Hour)

	w := doReq(t, r, "DELETE", "/api/v1/accounts/404", authToken(t, mgr), "")
	if w.Code != 200 {
		t.Fatalf("status=%d", w.Code)
	}
	if got := w.Body.String(); !strings.Contains(got, "40400") {
		t.Fatalf("want code 40400, got %s", got)
	}
}
