package adminapi

// 产品↔下发模板绑定 handler 契约:GET 空绑定 / PUT 绑定 / DELETE 解绑 / 参数校验。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

func bindingGet(t *testing.T, r *gin.Engine, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestOfferProvisionBindingHandlers(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	gin.SetMode(gin.TestMode)
	tok := authToken(t, mgr)

	t.Run("GET 未绑定返回空绑定", func(t *testing.T) {
		r := gin.New()
		Register(r, &app.Application{User: &fakeUser{permOk: true}, Provision: &fakeProvision{}}, mgr)
		w := bindingGet(t, r, "/api/admin/v1/products/101/provision-binding", tok)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var body struct {
			Code int `json:"code"`
			Data struct {
				OfferID    int64 `json:"offerId"`
				TemplateID int64 `json:"templateId"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Code != 0 || body.Data.OfferID != 101 || body.Data.TemplateID != 0 {
			t.Fatalf("body=%s", w.Body.String())
		}
	})

	t.Run("PUT 绑定套餐→模板", func(t *testing.T) {
		f := &fakeProvision{}
		r := gin.New()
		Register(r, &app.Application{User: &fakeUser{permOk: true}, Provision: f}, mgr)
		w := putJSONAuth(t, r, "/api/admin/v1/products/101/provision-binding", `{"templateId":142,"remark":"家庭宽带100M"}`, tok)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if f.bindRec == nil || f.bindRec.offerID != 101 || f.bindRec.templateID != 142 {
			t.Fatalf("bindRec=%+v", f.bindRec)
		}
	})

	t.Run("PUT 缺 templateId 拒绝", func(t *testing.T) {
		r := gin.New()
		Register(r, &app.Application{User: &fakeUser{permOk: true}, Provision: &fakeProvision{}}, mgr)
		w := putJSONAuth(t, r, "/api/admin/v1/products/101/provision-binding", `{}`, tok)
		var body struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Code != 42200 {
			t.Fatalf("want code=42200, body=%s", w.Body.String())
		}
	})

	t.Run("DELETE 解绑", func(t *testing.T) {
		f := &fakeProvision{}
		r := gin.New()
		Register(r, &app.Application{User: &fakeUser{permOk: true}, Provision: f}, mgr)
		w := deleteJSONAuth(r, "/api/admin/v1/products/101/provision-binding", tok)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if f.delRec != 101 {
			t.Fatalf("delRec=%d", f.delRec)
		}
	})

	t.Run("GET /provision-bindings 清单", func(t *testing.T) {
		r := gin.New()
		Register(r, &app.Application{User: &fakeUser{permOk: true}, Provision: &fakeProvision{}}, mgr)
		w := bindingGet(t, r, "/api/admin/v1/provision-bindings", tok)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	})
}
