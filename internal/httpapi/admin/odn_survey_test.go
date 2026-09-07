package adminapi

// 勘测任务 admin handler 测试(W7):列表信封/详情聚合/创建审计路径。

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
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

type fakeSurveyODN struct {
	odn.ODNService
	tasks     []odn.SurveyTask
	task      *odn.SurveyTask
	reports   []odn.SurveyReport
	created   *odn.SurveyCreateInput
	assigned  [2]int64
	cancelled bool
}

func (f *fakeSurveyODN) ListSurveys(_ context.Context, _ string, _ int64, _ int) ([]odn.SurveyTask, error) {
	return f.tasks, nil
}

func (f *fakeSurveyODN) CreateSurvey(_ context.Context, in odn.SurveyCreateInput) (*odn.SurveyTask, error) {
	f.created = &in
	return &odn.SurveyTask{ID: 9, TaskNo: "SV-20260907-00001", Title: in.Title, Status: odn.SurveyPending}, nil
}

func (f *fakeSurveyODN) GetSurvey(_ context.Context, _ int64) (*odn.SurveyTask, []odn.SurveyReport, error) {
	return f.task, f.reports, nil
}

func (f *fakeSurveyODN) AssignSurvey(_ context.Context, id, workerID int64) error {
	f.assigned = [2]int64{id, workerID}
	return nil
}

func (f *fakeSurveyODN) CancelSurvey(_ context.Context, _ int64) error {
	f.cancelled = true
	return nil
}

func surveyRouter(f *fakeSurveyODN) *gin.Engine {
	r := gin.New()
	a := &app.Application{User: &fakeUser{permOk: true}, ODN: f}
	mgr := auth.NewManager("test-secret", time.Hour)
	g := r.Group("/api/admin/v1", middleware.Authn(mgr, auth.AudAdmin))
	registerODNRoutes(g, a)
	return r
}

func surveyDo(r *gin.Engine, method, path, body string) map[string]any {
	tok, _ := auth.NewManager("test-secret", time.Hour).Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	out["_status"] = w.Code
	return out
}

func TestODNSurveyHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("创建勘测任务", func(t *testing.T) {
		f := &fakeSurveyODN{}
		res := surveyDo(surveyRouter(f), http.MethodPost, "/api/admin/v1/odn/surveys",
			`{"title":"网格A勘测","description":"主干覆盖","assignedWorkerId":6,"gridCode":3}`)
		if res["code"].(float64) != 0 {
			t.Fatalf("res=%v", res)
		}
		if f.created == nil || f.created.AssignedWorkerID != 6 || f.created.GridCode != 3 {
			t.Fatalf("input=%+v", f.created)
		}
	})

	t.Run("详情聚合 task+reports", func(t *testing.T) {
		f := &fakeSurveyODN{task: &odn.SurveyTask{ID: 1, TaskNo: "SV-1"},
			reports: []odn.SurveyReport{{ID: 5, Suggestion: odn.SuggestNeedNewFacility}}}
		res := surveyDo(surveyRouter(f), http.MethodGet, "/api/admin/v1/odn/surveys/1", "")
		d, _ := res["data"].(map[string]any)
		if d == nil || d["task"] == nil || d["reports"] == nil {
			t.Fatalf("res=%v", res)
		}
	})

	t.Run("指派与取消走审计路径", func(t *testing.T) {
		f := &fakeSurveyODN{}
		r := surveyRouter(f)
		if res := surveyDo(r, http.MethodPost, "/api/admin/v1/odn/surveys/1/assign", `{"workerId":6}`); res["code"].(float64) != 0 {
			t.Fatalf("assign res=%v", res)
		}
		if f.assigned != [2]int64{1, 6} {
			t.Fatalf("assigned=%v", f.assigned)
		}
		if res := surveyDo(r, http.MethodPost, "/api/admin/v1/odn/surveys/1/cancel", ""); res["code"].(float64) != 0 {
			t.Fatalf("cancel res=%v", res)
		}
		if !f.cancelled {
			t.Fatal("cancel not invoked")
		}
	})
}
