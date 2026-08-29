package adminapi

// 开单内联建址端点测试:门禁/契约透传/参数校验/冲突语义/兜底留痕。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

const inlineBody = `{"customerId":9,"city":"马尼拉市","district":"奎松区","street":"幸福街道",` +
	`"compound":"阳光小区","building":"3号楼","backfillCustomer":true}`

func postInline(t *testing.T, r http.Handler, tok string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/orders/address", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// inlineRouter Register 全量路由,注入 Notify 桩。
func inlineRouter(u *fakeUser, n notify.Service, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{User: u, Notify: n}, mgr)
	return r
}

// fakeNotify 桩 notify.Service:仅捕获 Emit 入参。
type fakeNotify struct {
	notify.Service
	emitted []notify.Input
}

func (f *fakeNotify) Emit(_ context.Context, in notify.Input) error {
	f.emitted = append(f.emitted, in)
	return nil
}

// TestOrderInlineAddressPermission menu:order 门禁:无权限 403 且不触达域层。
func TestOrderInlineAddressPermission(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	u := &fakeUser{permOk: false}
	r := inlineRouter(u, nil, mgr)
	w := postInline(t, r, authToken(t, mgr), inlineBody)
	if !strings.Contains(w.Body.String(), `"code":403`) {
		t.Fatalf("want 403, body=%s", w.Body.String())
	}
	if u.chainIn.CustomerID != 0 {
		t.Fatalf("域层不应被触达: %+v", u.chainIn)
	}
}

// TestOrderInlineAddressContract 契约:五级透传 + 回执字段映射。
func TestOrderInlineAddressContract(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	u := &fakeUser{permOk: true, chainRes: user.InlineAddressResult{
		AddressID: 502, FullPath: "manila.quesong.xingfu.yangguang.n_3f2a",
		FullPathNames: "马尼拉市 / 奎松区 / 幸福街道 / 阳光小区 / 3号楼",
		LegalEntityID: 2, RegionPath: "root.luzon",
		NeedsReview: []user.InlineAddressNode{{ID: 498, Level: 1, Name: "马尼拉市"}},
		Backfilled:  true,
	}}
	r := inlineRouter(u, nil, mgr)
	w := postInline(t, r, authToken(t, mgr), inlineBody)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"addressId":502`) {
		t.Fatalf("resp=%d %s", w.Code, w.Body.String())
	}
	in := u.chainIn
	if in.CustomerID != 9 || !in.BackfillCustomer || len(in.Levels) != 5 ||
		in.Levels[0].Name != "马尼拉市" || in.Levels[4].Name != "3号楼" || in.Levels[4].Level != 5 {
		t.Fatalf("入参透传失真: %+v", in)
	}
	for _, kw := range []string{`"legalEntityId":2`, `"fallback":false`, `"backfilled":true`, `"needsReview"`} {
		if !strings.Contains(w.Body.String(), kw) {
			t.Fatalf("缺字段 %s: %s", kw, w.Body.String())
		}
	}
}

// TestOrderInlineAddressValidation building 缺失 → 42200,不触达域层。
func TestOrderInlineAddressValidation(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	u := &fakeUser{permOk: true}
	r := inlineRouter(u, nil, mgr)
	w := postInline(t, r, authToken(t, mgr), `{"customerId":9,"building":""}`)
	if !strings.Contains(w.Body.String(), "42200") {
		t.Fatalf("want 42200, body=%s", w.Body.String())
	}
	if u.chainIn.CustomerID != 0 {
		t.Fatalf("校验失败不应触达域层")
	}
}

// TestOrderInlineAddressDuplicate 同父同名冲突:user.ErrDuplicate → 40900 明确语义。
func TestOrderInlineAddressDuplicate(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	u := &fakeUser{permOk: true, chainErr: user.ErrDuplicate}
	r := inlineRouter(u, nil, mgr)
	w := postInline(t, r, authToken(t, mgr), inlineBody)
	if !strings.Contains(w.Body.String(), "40900") {
		t.Fatalf("want 40900, body=%s", w.Body.String())
	}
}

// TestOrderInlineAddressCustomerMissing 客户不存在 → 40400。
func TestOrderInlineAddressCustomerMissing(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	u := &fakeUser{permOk: true, chainErr: user.ErrNotFound}
	r := inlineRouter(u, nil, mgr)
	w := postInline(t, r, authToken(t, mgr), inlineBody)
	if !strings.Contains(w.Body.String(), "40400") {
		t.Fatalf("want 40400, body=%s", w.Body.String())
	}
}

// TestOrderInlineAddressFallback 兜底态:fallback=true + 归属修正 P1 待办(幂等键=path,DueHours=4)。
func TestOrderInlineAddressFallback(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	fn := &fakeNotify{}
	u := &fakeUser{permOk: true, chainRes: user.InlineAddressResult{
		AddressID: 502, FullPath: "manila.queson.xingfu.yangguang.n_3f2a",
		FullPathNames: "马尼拉市 / 奎松区 / 幸福街道 / 阳光小区 / 3号楼",
		LegalEntityID: 1, RegionPath: "root", Fallback: true,
	}}
	w := postInline(t, inlineRouter(u, fn, mgr), authToken(t, mgr), inlineBody)
	if !strings.Contains(w.Body.String(), `"fallback":true`) {
		t.Fatalf("want fallback=true, body=%s", w.Body.String())
	}
	if len(fn.emitted) != 1 {
		t.Fatalf("待办未生成: %+v", fn.emitted)
	}
	in := fn.emitted[0]
	if in.Category != notify.CategoryTodo || in.Level != notify.LevelUrgent ||
		in.RefType != "order-inline-addr" || in.RefID != u.chainRes.FullPath || in.DueHours != 4 {
		t.Fatalf("待办参数失真: %+v", in)
	}
}

// TestOrderInlineAddressNoFallbackNoTodo 正常归属:不产生待办。
func TestOrderInlineAddressNoFallbackNoTodo(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	fn := &fakeNotify{}
	u := &fakeUser{permOk: true, chainRes: user.InlineAddressResult{AddressID: 1, LegalEntityID: 2}}
	w := postInline(t, inlineRouter(u, fn, mgr), authToken(t, mgr), inlineBody)
	if w.Code != http.StatusOK || len(fn.emitted) != 0 {
		t.Fatalf("resp=%s emitted=%+v", w.Body.String(), fn.emitted)
	}
}
