package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeOnboarding 桩 OnboardingService / RealNameService。
type fakeOnboarding struct {
	submitted  worker.Registration
	submittedN int
	approved   []int64
	rejected   []int64
	rejectNote string
	list       []worker.Registration
	rnSub      worker.WorkerRealNameVerification
	rnVerify   []worker.WorkerRealNameVerification
	latest     *worker.WorkerRealNameVerification
	approveErr error
}

func (f *fakeOnboarding) Submit(_ context.Context, reg worker.Registration) (int64, error) {
	f.submitted = reg
	f.submittedN++
	return int64(f.submittedN), nil
}
func (f *fakeOnboarding) ListRegistrations(_ context.Context, _ string) ([]worker.Registration, error) {
	return f.list, nil
}
func (f *fakeOnboarding) Approve(_ context.Context, id, _ int64) (int64, error) {
	if f.approveErr != nil {
		return 0, f.approveErr
	}
	f.approved = append(f.approved, id)
	return 99, nil
}
func (f *fakeOnboarding) Reject(_ context.Context, id, _ int64, note string) error {
	f.rejected = append(f.rejected, id)
	f.rejectNote = note
	return nil
}
func (f *fakeOnboarding) SubmitRealName(_ context.Context, v worker.WorkerRealNameVerification) (int64, error) {
	f.rnSub = v
	return 7, nil
}
func (f *fakeOnboarding) GetLatest(_ context.Context, _ int64) (*worker.WorkerRealNameVerification, error) {
	return f.latest, nil
}
func (f *fakeOnboarding) Verify(_ context.Context, wid int64, result, _ string, _ int64) error {
	f.rnVerify = append(f.rnVerify, worker.WorkerRealNameVerification{WorkerID: wid, Result: result})
	return nil
}

func newOnboardingTestRouter(f *fakeOnboarding) (*gin.Engine, *auth.Manager) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mgr := auth.NewManager("test-secret", time.Hour)
	RegisterRoutes(r, &Application{
		User:             &fakeUser{permOk: true},
		Worker:           &fakeWorkerOps{},
		WorkerOnboarding: f,
		WorkerRealName:   f,
	}, mgr)
	return r, mgr
}

func TestWorkerOnboarding_PublicSubmit(t *testing.T) {
	f := &fakeOnboarding{}
	r, _ := newOnboardingTestRouter(f)

	// 公开端点:无认证头也应 200。
	req := httptest.NewRequest(http.MethodPost, "/api/v1/worker-registrations",
		strings.NewReader(`{"name":"王师傅","phone":"13800000001","idCardNo":"110101199001011234","groupId":6,"regionId":4}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Fatalf("code=%d", resp.Code)
	}
	if f.submitted.Name != "王师傅" || f.submitted.GroupID != 6 {
		t.Fatalf("submitted=%+v", f.submitted)
	}
}

func TestWorkerOnboarding_ApproveAndVerify(t *testing.T) {
	f := &fakeOnboarding{}
	r, mgr := newOnboardingTestRouter(f)
	tok := authToken(t, mgr)

	// 审核通过
	w := postBodyAuth(t, r, "/api/v1/worker-registrations/5/approve", `{}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("approve code=%d body=%s", w.Code, w.Body.String())
	}
	if len(f.approved) != 1 || f.approved[0] != 5 {
		t.Fatalf("approved=%v", f.approved)
	}

	// 提交实名
	w = postBodyAuth(t, r, "/api/v1/workers/9/real-name",
		`{"realName":"王师傅","idCardNo":"110101199001011234","method":"证件OCR"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("realname submit code=%d body=%s", w.Code, w.Body.String())
	}
	if f.rnSub.WorkerID != 9 || f.rnSub.Method != "证件OCR" {
		t.Fatalf("rnSub=%+v", f.rnSub)
	}

	// 后台核验
	w = postBodyAuth(t, r, "/api/v1/workers/9/real-name/verify", `{"result":"PASS"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("verify code=%d body=%s", w.Code, w.Body.String())
	}
	if len(f.rnVerify) != 1 || f.rnVerify[0].Result != worker.RealNamePass {
		t.Fatalf("rnVerify=%+v", f.rnVerify)
	}
}

func TestWorkerOnboarding_Reject(t *testing.T) {
	f := &fakeOnboarding{}
	r, mgr := newOnboardingTestRouter(f)
	tok := authToken(t, mgr)

	w := postBodyAuth(t, r, "/api/v1/worker-registrations/8/reject", `{"note":"证件存疑"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("reject code=%d body=%s", w.Code, w.Body.String())
	}
	if len(f.rejected) != 1 || f.rejected[0] != 8 || f.rejectNote != "证件存疑" {
		t.Fatalf("rejected=%v note=%s", f.rejected, f.rejectNote)
	}
}
