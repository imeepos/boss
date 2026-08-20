package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/audit"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// fakeAuditWriter 记录写入事件,满足 audit.Writer。
type fakeAuditWriter struct {
	events []audit.Event
}

func (f *fakeAuditWriter) Write(_ context.Context, e audit.Event) error {
	f.events = append(f.events, e)
	return nil
}

func (f *fakeAuditWriter) List(context.Context, audit.Query) ([]audit.Entry, error) { return nil, nil }

// stubUserSvc 内嵌接口零值满足 user.Service,仅覆写 GetProfile。
type stubUserSvc struct {
	user.Service
	prof *user.Profile
	err  error
}

func (s stubUserSvc) GetProfile(context.Context, int64) (*user.Profile, error) { return s.prof, s.err }

// stubWorkerSvc 仅覆写 GetWorker。
type stubWorkerSvc struct {
	worker.WorkerService
	w   *worker.Worker
	err error
}

func (s stubWorkerSvc) GetWorker(context.Context, int64) (*worker.Worker, error) { return s.w, s.err }

// stubCustomerSvc 仅覆写 Get。
type stubCustomerSvc struct {
	customer.CustomerService
	c   *customer.Customer
	err error
}

func (s stubCustomerSvc) Get(context.Context, int64) (*customer.Customer, error) { return s.c, s.err }

func testCtx() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	return c, w
}

// envelope 统一响应结构,用于断言 Respond 写出的 JSON。
type envelope struct {
	Code apitypes.Code `json:"code"`
	Msg  string        `json:"msg"`
	Data any           `json:"data"`
}

// respondCode 执行 fn 并解出响应 code。
func respondCode(fn func(*gin.Context)) apitypes.Code {
	c, w := testCtx()
	fn(c)
	var e envelope
	if err := json.Unmarshal(w.Body.Bytes(), &e); err != nil {
		panic(err)
	}
	return e.Code
}

func TestRespond(t *testing.T) {
	c, w := testCtx()
	Respond(c, apitypes.CodeOK, map[string]int{"id": 7})
	if c.Writer.Status() != http.StatusOK {
		t.Fatalf("status = %d", c.Writer.Status())
	}
	var e envelope
	if err := json.Unmarshal(w.Body.Bytes(), &e); err != nil {
		t.Fatal(err)
	}
	if e.Code != apitypes.CodeOK || e.Data == nil {
		t.Fatalf("envelope = %+v", e)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func TestClaimsAccountID(t *testing.T) {
	c, _ := testCtx()
	if got := ClaimsAccountID(c); got != 0 {
		t.Fatalf("no claims: got %d", got)
	}
	c.Set(middleware.CtxClaims, "not-claims")
	if got := ClaimsAccountID(c); got != 0 {
		t.Fatalf("wrong type: got %d", got)
	}
	c.Set(middleware.CtxClaims, &auth.Claims{AccountID: 42})
	if got := ClaimsAccountID(c); got != 42 {
		t.Fatalf("claims: got %d", got)
	}
}

func TestRecordAudit(t *testing.T) {
	c, _ := testCtx()
	RecordAudit(nil, c, "a", "order", "1", nil)            // nil app
	RecordAudit(&app.Application{}, c, "a", "o", "1", nil) // nil writer

	fw := &fakeAuditWriter{}
	a := &app.Application{Audit: fw}
	RecordAudit(a, c, "create", "order", "9", map[string]any{"k": 1})
	if len(fw.events) != 1 {
		t.Fatalf("events = %d", len(fw.events))
	}
	e := fw.events[0]
	if e.Action != "create" || e.TargetType != "order" || e.TargetID != "9" || e.IP == "" {
		t.Fatalf("event = %+v", e)
	}
}

func TestAPIKeySubjectResolver(t *testing.T) {
	ctx := context.Background()
	a := &app.Application{
		User:     stubUserSvc{prof: &user.Profile{Username: "alice", RoleCode: "ADMIN"}},
		Worker:   stubWorkerSvc{w: &worker.Worker{Name: "bob"}},
		Customer: stubCustomerSvc{c: &customer.Customer{Name: "acme"}},
	}
	r := APIKeySubjectResolver(a)

	name, role, err := r(ctx, "account", 1)
	if err != nil || name != "alice" || role != "ADMIN" {
		t.Fatalf("account: %q %q %v", name, role, err)
	}
	name, role, err = r(ctx, "worker", 1)
	if err != nil || name != "bob" || role != "worker" {
		t.Fatalf("worker: %q %q %v", name, role, err)
	}
	name, role, err = r(ctx, "customer", 1)
	if err != nil || name != "acme" || role != "customer" {
		t.Fatalf("customer: %q %q %v", name, role, err)
	}
	name, role, err = r(ctx, "unknown", 1)
	if err != nil || name != "" || role != "" {
		t.Fatalf("unknown: %q %q %v", name, role, err)
	}

	aErr := &app.Application{
		User:     stubUserSvc{err: errors.New("u")},
		Worker:   stubWorkerSvc{err: errors.New("w")},
		Customer: stubCustomerSvc{err: errors.New("c")},
	}
	rErr := APIKeySubjectResolver(aErr)
	if _, _, err := rErr(ctx, "account", 1); err == nil {
		t.Fatal("account err expected")
	}
	if _, _, err := rErr(ctx, "worker", 1); err == nil {
		t.Fatal("worker err expected")
	}
	if _, _, err := rErr(ctx, "customer", 1); err == nil {
		t.Fatal("customer err expected")
	}
}
