package workerapi

// 勘测任务/施工进度师傅端 handler 测试(W7):身份注入/接单/回填/进度上报代次。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
)

type fakeWorkerODN struct {
	odn.ODNService
	visible      []odn.SurveyTask
	reports      []odn.SurveyReport
	accepted     [2]int64
	lastReport   *odn.SurveyReport
	lastProgress *odn.ProgressEntry
	created      bool
}

func (f *fakeWorkerODN) ListSurveysForWorker(_ context.Context, _ int64, _ string, _ int) ([]odn.SurveyTask, error) {
	return f.visible, nil
}

func (f *fakeWorkerODN) GetSurvey(_ context.Context, _ int64) (*odn.SurveyTask, []odn.SurveyReport, error) {
	return &odn.SurveyTask{ID: 1, TaskNo: "SV-1", AssignedWorkerID: 7}, f.reports, nil
}

func (f *fakeWorkerODN) AcceptSurvey(_ context.Context, id, workerID int64) error {
	f.accepted = [2]int64{id, workerID}
	return nil
}

func (f *fakeWorkerODN) AddSurveyReport(_ context.Context, r odn.SurveyReport) (int64, bool, error) {
	ff := r
	f.lastReport = &ff
	return 11, true, nil
}

func (f *fakeWorkerODN) ListBuildingProjects(_ context.Context, _ int) ([]odn.Construction, error) {
	return []odn.Construction{{ID: 4, ProjNo: "C-1", Status: odn.CBuilding}}, nil
}

func (f *fakeWorkerODN) GetProjectWorkerView(_ context.Context, _ int64) (*odn.Construction, []odn.ItemProgress, error) {
	return &odn.Construction{ID: 4, Status: odn.CBuilding}, []odn.ItemProgress{{FacilityCode: "P01001"}}, nil
}

func (f *fakeWorkerODN) RecordProgress(_ context.Context, p odn.ProgressEntry) (int64, bool, error) {
	ff := p
	f.lastProgress = &ff
	return 21, true, nil
}

func surveyWorkerRouter(f *fakeWorkerODN) *gin.Engine {
	r := gin.New()
	a := &app.Application{ODN: f}
	g := r.Group("/api/worker/v1", func(c *gin.Context) {
		c.Set(ctxPortalWorkerID, int64(7))
		c.Set(ctxPortalWorkerName, "张师傅")
		c.Next()
	})
	registerWorkerSurveyRoutes(g, a)
	registerWorkerConstructionRoutes(g, a)
	return r
}

func workerSurveyDo(r *gin.Engine, method, path, body string) map[string]any {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	out["_status"] = w.Code
	return out
}

func TestWorkerSurveyHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("可见集列表", func(t *testing.T) {
		f := &fakeWorkerODN{visible: []odn.SurveyTask{{ID: 1, TaskNo: "SV-1", Status: odn.SurveyPending}}}
		res := workerSurveyDo(surveyWorkerRouter(f), http.MethodGet, "/api/worker/v1/surveys", "")
		if res["code"].(float64) != 0 {
			t.Fatalf("res=%v", res)
		}
	})

	t.Run("接单带师傅身份", func(t *testing.T) {
		f := &fakeWorkerODN{}
		res := workerSurveyDo(surveyWorkerRouter(f), http.MethodPost, "/api/worker/v1/surveys/1/accept", "")
		if res["code"].(float64) != 0 || f.accepted != [2]int64{1, 7} {
			t.Fatalf("res=%v accepted=%v", res, f.accepted)
		}
	})

	t.Run("回填走 append-only 幂等路径", func(t *testing.T) {
		f := &fakeWorkerODN{}
		body := `{"lat":14.6,"lng":120.98,"facilityNote":"杆路完好","suggestion":"CAN_INSTALL","photoIds":[3],"clientMsgId":"w7-msg-0001"}`
		res := workerSurveyDo(surveyWorkerRouter(f), http.MethodPost, "/api/worker/v1/surveys/1/reports", body)
		if res["code"].(float64) != 0 || f.lastReport == nil || f.lastReport.WorkerID != 7 {
			t.Fatalf("res=%v report=%+v", res, f.lastReport)
		}
	})

	t.Run("进度上报带 WORKER 代次", func(t *testing.T) {
		f := &fakeWorkerODN{}
		body := `{"facilityCode":"P01001","doneQty":2,"lat":14.6,"lng":120.98,"clientMsgId":"w7-prog-0001"}`
		res := workerSurveyDo(surveyWorkerRouter(f), http.MethodPost, "/api/worker/v1/constructions/4/progress", body)
		if res["code"].(float64) != 0 {
			t.Fatalf("res=%v", res)
		}
		if f.lastProgress == nil || f.lastProgress.ReporterType != odn.ProgressReporterWorker || f.lastProgress.ReportedBy != 7 {
			t.Fatalf("progress=%+v", f.lastProgress)
		}
	})
}
