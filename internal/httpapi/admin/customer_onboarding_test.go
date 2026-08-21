package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	userapi "github.com/ymm-001/boss/internal/httpapi/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/realid"
)

// fakeCustOnboard 桩 OnboardingService / 实名核验接口。
type fakeCustOnboard struct {
	submitted  customer.Registration
	submittedN int
	approved   []int64
	rejected   []int64
	rejectNote string
	list       []customer.Registration
	rnSub      customer.CustomerRealNameVerification
	rnVerify   []customer.CustomerRealNameVerification
	latest     *customer.CustomerRealNameVerification
	apxErr     error
}

func (f *fakeCustOnboard) Submit(_ context.Context, reg customer.Registration) (int64, error) {
	f.submitted = reg
	f.submittedN++
	return int64(f.submittedN), nil
}
func (f *fakeCustOnboard) ListRegistrations(_ context.Context, _ string) ([]customer.Registration, error) {
	return f.list, nil
}
func (f *fakeCustOnboard) Approve(_ context.Context, id, _ int64) (int64, error) {
	if f.apxErr != nil {
		return 0, f.apxErr
	}
	f.approved = append(f.approved, id)
	return 88, nil
}
func (f *fakeCustOnboard) Reject(_ context.Context, id, _ int64, note string) error {
	f.rejected = append(f.rejected, id)
	f.rejectNote = note
	return nil
}
func (f *fakeCustOnboard) SubmitRealName(_ context.Context, v customer.CustomerRealNameVerification) (int64, error) {
	f.rnSub = v
	return 7, nil
}
func (f *fakeCustOnboard) GetLatest(_ context.Context, _ int64) (*customer.CustomerRealNameVerification, error) {
	return f.latest, nil
}
func (f *fakeCustOnboard) Verify(_ context.Context, id int64, result, reason, _ string, _ int64) error {
	f.rnVerify = append(f.rnVerify, customer.CustomerRealNameVerification{CustomerID: id, Result: result})
	return nil
}
func (f *fakeCustOnboard) ListVerifications(context.Context, int64) ([]customer.RealNameVerification, error) {
	return nil, nil
}
func (f *fakeCustOnboard) AppendVerification(context.Context, customer.RealNameVerification) (int64, error) {
	return 0, nil
}

func newCustOnboardRouter(f *fakeCustOnboard, rid ...realid.Verifier) (*gin.Engine, *auth.Manager) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("s", time.Hour)
	r := gin.New()
	ja := &app.Application{
		User:               &fakeUser{permOk: true},
		CustomerOnboarding: f,
		CustomerRealName:   f,
		RealName:           f,
	}
	if len(rid) > 0 {
		ja.RealID = rid[0]
	}
	Register(r, ja, mgr)
	userapi.Register(r, ja, mgr)
	return r, mgr
}

// TestCustomerOnboarding_PublicSubmit 客户自助注册公开端点(无认证头)。
func TestCustomerOnboarding_PublicSubmit(t *testing.T) {
	f := &fakeCustOnboard{}
	r, _ := newCustOnboardRouter(f)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/user/v1/customer-registrations",
		strings.NewReader(`{"name":"张先生","phone":"13800001234","idCardNo":"110101199001011234","legalEntityId":1,"addressId":100,"regionId":4}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 || f.submitted.Name != "张先生" || f.submitted.AddressID != 100 {
		t.Fatalf("resp=%+v submitted=%+v", resp, f.submitted)
	}
}

// TestCustomerOnboarding_ApproveRejectVerify 审核+实名核验闭环。
func TestCustomerOnboarding_ApproveRejectVerify(t *testing.T) {
	f := &fakeCustOnboard{}
	r, mgr := newCustOnboardRouter(f)
	tok := authToken(t, mgr)

	// 审核通过 → 建客户主档
	w := postBodyAuth(t, r, "/api/admin/v1/customer-registrations/5/approve", `{}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("approve code=%d body=%s", w.Code, w.Body.String())
	}
	if len(f.approved) != 1 || f.approved[0] != 5 {
		t.Fatalf("approved=%v", f.approved)
	}

	// 提交实名 → 落 PENDING
	w = postBodyAuth(t, r, "/api/admin/v1/customers/88/real-name",
		`{"realName":"张先生","idCardNo":"110101199001011234","method":"证件OCR"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("realname submit code=%d body=%s", w.Code, w.Body.String())
	}
	if f.rnSub.CustomerID != 88 || f.rnSub.Method != "证件OCR" || f.rnSub.Result != customer.RealNamePending {
		t.Fatalf("rnSub=%+v", f.rnSub)
	}

	// 后台核验 PASS
	w = postBodyAuth(t, r, "/api/admin/v1/customers/88/real-name/verify", `{"result":"PASS"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("verify code=%d body=%s", w.Code, w.Body.String())
	}
	if len(f.rnVerify) != 1 || f.rnVerify[0].Result != customer.RealNamePass {
		t.Fatalf("rnVerify=%+v", f.rnVerify)
	}

	// 审核驳回
	w = postBodyAuth(t, r, "/api/admin/v1/customer-registrations/8/reject", `{"note":"地址信息存疑"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("reject code=%d body=%s", w.Code, w.Body.String())
	}
	if len(f.rejected) != 1 || f.rejected[0] != 8 || f.rejectNote != "地址信息存疑" {
		t.Fatalf("rejected=%v note=%s", f.rejected, f.rejectNote)
	}
}

// TestCustomerOnboarding_InvalidVerify 非法核验结果(非 PASS/FAIL)拒。
func TestCustomerOnboarding_InvalidVerify(t *testing.T) {
	f := &fakeCustOnboard{}
	r, mgr := newCustOnboardRouter(f)
	tok := authToken(t, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/customers/1/real-name/verify", `{"result":"MAYBE"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d", w.Code)
	}
	var resp struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code == 0 || len(f.rnVerify) != 0 {
		t.Fatalf("应拒绝非法核验: resp=%+v rnVerify=%v", resp, f.rnVerify)
	}
}

// TestCustomerOnboarding_List 审核队列按状态列出。
func TestCustomerOnboarding_List(t *testing.T) {
	f := &fakeCustOnboard{list: []customer.Registration{{ID: 1, Name: "张先生", Status: customer.RegStatusPending}}}
	r, mgr := newCustOnboardRouter(f)
	w := getJSON(t, r, "/api/admin/v1/customer-registrations?status=PENDING", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []customer.Registration `json:"items"`
		}
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 || len(resp.Data.Items) != 1 || resp.Data.Items[0].Name != "张先生" {
		t.Fatalf("resp=%+v", resp)
	}
}

// TestCustomerOnboarding_ApproveConflict 冲突错误 → 409xx 错误码(非 0)。
func TestCustomerOnboarding_ApproveConflict(t *testing.T) {
	f := &fakeCustOnboard{}
	f.apxErr = customer.ErrRegistrationConflict
	r, mgr := newCustOnboardRouter(f)
	w := postBodyAuth(t, r, "/api/admin/v1/customer-registrations/5/approve", `{}`, authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d", w.Code)
	}
	var resp struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code == 0 {
		t.Fatalf("期望冲突错误码(非 0),resp=%+v", resp)
	}
}

// fakeRealID 桩二要素核验通道。
type fakeRealID struct {
	decision string
	err      error
	calls    int
}

func (f *fakeRealID) Verify(_ context.Context, _, _ string) (string, error) {
	f.calls++
	return f.decision, f.err
}

// TestCustomerOnboarding_AutoRealID 二要素通道启用:提交即自动核验,结论落 verifications。
func TestCustomerOnboarding_AutoRealID(t *testing.T) {
	f := &fakeCustOnboard{}
	rid := &fakeRealID{decision: realid.Pass}
	r, mgr := newCustOnboardRouter(f, rid)
	tok := authToken(t, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/customers/88/real-name",
		`{"realName":"张先生","idCardNo":"110101199001011234","method":"证件OCR"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Result string `json:"result"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Result != realid.Pass || rid.calls != 1 {
		t.Fatalf("result=%s calls=%d", resp.Data.Result, rid.calls)
	}
	if len(f.rnVerify) != 1 || f.rnVerify[0].Result != customer.RealNamePass {
		t.Fatalf("rnVerify=%+v", f.rnVerify)
	}
}

// TestCustomerOnboarding_AutoRealIDFail 核验不一致 → FAIL;通道报错 → PENDING 人工兜底。
func TestCustomerOnboarding_AutoRealIDFail(t *testing.T) {
	f := &fakeCustOnboard{}
	r, mgr := newCustOnboardRouter(f, &fakeRealID{decision: realid.Fail})
	tok := authToken(t, mgr)
	w := postBodyAuth(t, r, "/api/admin/v1/customers/88/real-name",
		`{"realName":"张先生","idCardNo":"110101199001011234","method":"证件OCR"}`, tok)
	var resp struct {
		Data struct {
			Result string `json:"result"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Result != realid.Fail || len(f.rnVerify) != 1 {
		t.Fatalf("result=%s rnVerify=%+v", resp.Data.Result, f.rnVerify)
	}

	// 通道故障:不阻塞提交,保持 PENDING 且不落核验记录。
	f2 := &fakeCustOnboard{}
	r2, mgr2 := newCustOnboardRouter(f2, &fakeRealID{err: errors.New("boom")})
	w2 := postBodyAuth(t, r2, "/api/admin/v1/customers/88/real-name",
		`{"realName":"张先生","idCardNo":"110101199001011234","method":"证件OCR"}`, authToken(t, mgr2))
	if w2.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w2.Code, w2.Body.String())
	}
	var resp2 struct {
		Data struct {
			Result string `json:"result"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.Data.Result != customer.RealNamePending || len(f2.rnVerify) != 0 {
		t.Fatalf("result=%s rnVerify=%+v", resp2.Data.Result, f2.rnVerify)
	}
}
