// 回归:实名提交 → 消息中心落 todo 待办(Notify.Emit,幂等键 realname/customer/{id});
// 人工审核终态 → 办结待办 + 客户站内消息回执(Notify.Resolve + Portal.PutMessage)。
package adminapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

func newRealnameNotifyRouter(f *fakeCustOnboard, ns notify.Service, ps portal.Service) (*gin.Engine, *auth.Manager) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("s", time.Hour)
	r := gin.New()
	ja := &app.Application{
		User:             &fakeUser{permOk: true},
		CustomerRealName: f,
		RealName:         f,
		Notify:           ns,
		Portal:           ps,
	}
	Register(r, ja, mgr)
	return r, mgr
}

func TestCustomerRealnameSubmitEmitsTodo(t *testing.T) {
	ns := notify.NewMemStore()
	r, mgr := newRealnameNotifyRouter(&fakeCustOnboard{}, ns, nil)
	tok := authToken(t, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/customers/88/real-name",
		`{"realName":"张先生","idCardNo":"110101199001011234","method":"证件OCR"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	items, total, err := ns.List(context.Background(), "sysadmin", 1, notify.Filter{})
	if err != nil || total != 1 {
		t.Fatalf("list total=%d err=%v", total, err)
	}
	it := items[0]
	if it.Category != notify.CategoryTodo || it.RefType != "realname" || it.RefID != "customer/88" {
		t.Fatalf("item=%+v", it)
	}
	if it.Link != "/base/realname-review" || it.Title == "" {
		t.Fatalf("link/title empty: %+v", it)
	}
}

func TestCustomerRealnameVerifyResolvesTodoAndNotifies(t *testing.T) {
	ns := notify.NewMemStore()
	ps := portal.NewMemory()
	r, mgr := newRealnameNotifyRouter(&fakeCustOnboard{}, ns, ps)
	tok := authToken(t, mgr)

	ctx := context.Background()
	_ = ns.Emit(ctx, notify.Input{Category: notify.CategoryTodo, Title: "实名待审核",
		RefType: "realname", RefID: "customer/88"})

	w := postBodyAuth(t, r, "/api/admin/v1/customers/88/real-name/verify",
		`{"result":"FAIL","reason":"证件照片模糊"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	items, _, err := ns.List(ctx, "sysadmin", 1, notify.Filter{})
	if err != nil || len(items) != 1 || !items[0].Resolved {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	msgs, _ := ps.Messages(ctx, 88)
	if len(msgs) != 1 {
		t.Fatalf("messages=%d", len(msgs))
	}
	p := msgs[0].Payload
	if p["category"] != "system" || p["title"] != "实名认证未通过" ||
		p["content"] != "您提交的实名认证未通过审核：证件照片模糊" {
		t.Fatalf("payload=%+v", p)
	}
}

func TestCustomerRealnameVerifyPassNotifiesPass(t *testing.T) {
	ps := portal.NewMemory()
	r, mgr := newRealnameNotifyRouter(&fakeCustOnboard{}, nil, ps)
	tok := authToken(t, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/customers/88/real-name/verify", `{"result":"PASS"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	msgs, _ := ps.Messages(context.Background(), 88)
	if len(msgs) != 1 || msgs[0].Payload["title"] != "实名认证已通过" {
		t.Fatalf("messages=%+v", msgs)
	}
}

// 回归:师傅实名审核终态 → worker_messages 回执(WARN/FAIL、INFO/PASS)。
func TestWorkerRealnameVerifyNotifiesWorkerMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("s", time.Hour)
	r := gin.New()
	wf := &fakeOnboarding{}
	led := &recordingLedger{fakeWorkerOps: &fakeWorkerOps{}}
	ja := &app.Application{
		User:           &fakeUser{permOk: true},
		WorkerRealName: wf,
		WorkerLedger:   led,
	}
	Register(r, ja, mgr)
	tok := authToken(t, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/verifications/worker/5/verify", `{"result":"FAIL"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if len(led.sent) != 1 || led.sent[0].Level != "WARN" || led.sent[0].WorkerID != 5 {
		t.Fatalf("sent=%+v", led.sent)
	}
	if led.sent[0].Title != "实名认证未通过" {
		t.Fatalf("title=%q", led.sent[0].Title)
	}
}

// recordingLedger 嵌入既有全桩,只覆写 SendMessage 记录下发内容。
type recordingLedger struct {
	*fakeWorkerOps
	sent []worker.Message
}

func (r *recordingLedger) SendMessage(_ context.Context, m worker.Message) (int64, error) {
	r.sent = append(r.sent, m)
	return int64(len(r.sent)), nil
}

// 回归:审核中心行内核验 subjectId 接受负数段合成客户(隔离空间 ID),仅拒 0/非整数;
// 修复前走通用 ParsePathParamInt64 的 `<=0` 校验,负数主体一律 42200,新注册合成客户无法过审。
func TestReviewCenterVerifyAcceptsSyntheticNegativeSubject(t *testing.T) {
	f := &fakeCustOnboard{}
	ns := notify.NewMemStore()
	r, mgr := newRealnameNotifyRouter(f, ns, nil)
	tok := authToken(t, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/verifications/customer/-9/verify", `{"result":"PASS"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if len(f.rnVerify) != 1 || f.rnVerify[0].CustomerID != -9 || f.rnVerify[0].Result != "PASS" {
		t.Fatalf("rnVerify=%+v", f.rnVerify)
	}

	w0 := postBodyAuth(t, r, "/api/admin/v1/verifications/customer/0/verify", `{"result":"PASS"}`, tok)
	if w0.Code != http.StatusOK || !strings.Contains(w0.Body.String(), "42200") {
		t.Fatalf("subjectId=0 应 42200: status=%d body=%s", w0.Code, w0.Body.String())
	}
}
