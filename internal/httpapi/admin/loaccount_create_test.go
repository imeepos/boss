package adminapi

// T2 AAA 建号三态契约:成功(200+CodeOK,缺省 POSTPAID+ACTIVE)/必填与白名单(42200)/
// loid 撞库与 offer 非 PUBLISHED(40900)。域侧校验由 aaa.LoAccountAdminService 承载,此处验证映射。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// fakeAaaCreate 桩 aaa.AaaService:内嵌接口,仅实现管理端建号。
type fakeAaaCreate struct {
	aaa.AaaService
	created aaa.LoAccount
	err     error
}

func (f *fakeAaaCreate) CreateLoAccountChecked(_ context.Context, a aaa.LoAccount) (int64, error) {
	f.created = a
	if f.err != nil {
		return 0, f.err
	}
	return 701, nil
}

func newLoAccountRouter(aa aaa.AaaService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("nas-test-secret", time.Hour) // 与 nasToken 同 secret,令牌互通
	r := gin.New()
	Register(r, &app.Application{User: &fakeUser{permOk: true}, Aaa: aa}, mgr)
	return r
}

func postLoAccount(t *testing.T, r *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()
	return doNasAuth(t, r, http.MethodPost, "/api/admin/v1/lo-accounts", nasToken(t), body)
}

func loCode(t *testing.T, w *httptest.ResponseRecorder) int {
	t.Helper()
	var out struct {
		Code int `json:"code"`
	}
	_ = json.NewDecoder(w.Body).Decode(&out)
	return out.Code
}

func TestPostLoAccounts(t *testing.T) {
	t.Run("建号成功:缺省 POSTPAID+ACTIVE", func(t *testing.T) {
		fa := &fakeAaaCreate{}
		r := newLoAccountRouter(fa)
		w := postLoAccount(t, r, `{"loid":"OWPAL19516","customerId":11,"offerId":5,"qosTemplateId":2}`)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if fa.created.BillingMode != aaa.BillingModePostpaid || fa.created.Status != string(aaa.StatusActive) {
			t.Fatalf("created=%+v", fa.created)
		}
	})

	t.Run("缺 qosTemplateId → 42200", func(t *testing.T) {
		r := newLoAccountRouter(&fakeAaaCreate{})
		w := postLoAccount(t, r, `{"loid":"L-1","customerId":11,"offerId":5}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("billingMode 白名单外 → 42200", func(t *testing.T) {
		r := newLoAccountRouter(&fakeAaaCreate{})
		w := postLoAccount(t, r, `{"loid":"L-2","customerId":11,"offerId":5,"qosTemplateId":2,"billingMode":"WEEKLY"}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("loid 撞库 → 40900", func(t *testing.T) {
		r := newLoAccountRouter(&fakeAaaCreate{err: aaa.ErrDuplicate})
		w := postLoAccount(t, r, `{"loid":"LOID-1","customerId":11,"offerId":5,"qosTemplateId":2}`)
		if code := loCode(t, w); code != int(apitypes.CodeConflict) {
			t.Fatalf("code=%d body=%s", code, w.Body.String())
		}
	})

	t.Run("offer 非 PUBLISHED → 40900", func(t *testing.T) {
		r := newLoAccountRouter(&fakeAaaCreate{err: aaa.ErrOfferNotPublished})
		w := postLoAccount(t, r, `{"loid":"L-3","customerId":11,"offerId":6,"qosTemplateId":2}`)
		if code := loCode(t, w); code != int(apitypes.CodeConflict) {
			t.Fatalf("code=%d", code)
		}
	})
}
