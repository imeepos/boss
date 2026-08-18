package app

// 用户端数据域路由表驱动 happy path:userdata.yaml 全部 42 操作逐一打通。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

func newUserdataRouter(ud *fakeUserdata, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, &Application{User: &fakeUser{permOk: true}, UserData: ud}, mgr)
	return r
}

// doReq 带鉴权令牌发起任意方法请求;body 可为空。
func doReq(t *testing.T, r *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rd *strings.Reader
	if body == "" {
		rd = strings.NewReader("")
	} else {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// assertOK 断言 envelope code=0。
func assertOK(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s: %v", w.Body.String(), err)
	}
	if body.Code != 0 {
		t.Fatalf("code=%d body=%s", body.Code, w.Body.String())
	}
	return body.Data
}

func TestUserdataEndpoints_HappyPath(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	ud := &fakeUserdata{row: map[string]any{"name": "王先生", "addonId": "ADD-01"}}
	r := newUserdataRouter(ud, mgr)
	tk := authToken(t, mgr)

	cases := []struct {
		method, path, body string
		wantCalled         string // 写路径目标 id 断言(可选)
	}{
		{"GET", "/api/v1/users", "", ""},
		{"GET", "/api/v1/users/9", "", ""},
		{"PUT", "/api/v1/users/9/account", `{"autoPay":true}`, ""},
		{"GET", "/api/v1/user-accounts", "", ""},
		{"GET", "/api/v1/user-addresses", "", ""},
		{"POST", "/api/v1/user-addresses", `{"customerId":9,"addrCode":"PH-MNL","contact":"王先生","phone":"138","detail":"XX路1号"}`, ""},
		{"GET", "/api/v1/user-plans", "", ""},
		{"POST", "/api/v1/user-plans", `{"customerId":9,"productId":3,"planName":"融合套餐"}`, ""},
		{"GET", "/api/v1/addons", "", ""},
		{"POST", "/api/v1/addons", `{"name":"加速包","price":1000}`, ""},
		{"PUT", "/api/v1/addons/ADD-01/toggle", "", "ADD-01"},
		{"GET", "/api/v1/addon-subscriptions", "", ""},
		{"POST", "/api/v1/addon-subscriptions", `{"customerId":9,"addonId":"ADD-01","action":"subscribe"}`, ""},
		{"GET", "/api/v1/user-notify-settings", "", ""},
		{"PUT", "/api/v1/user-notify-settings/9", `{"business":true,"marketing":false,"channel":"sms"}`, ""},
		{"GET", "/api/v1/user-faqs", "", ""},
		{"POST", "/api/v1/user-faqs", `{"category":"billing","question":"如何开发票?","answer":"自助开具","active":true}`, ""},
		{"PUT", "/api/v1/user-faqs/FAQ-01/toggle", "", "FAQ-01"},
		{"GET", "/api/v1/user-messages", "", ""},
		{"POST", "/api/v1/user-messages", `{"customerId":9,"title":"缴费提醒","content":"请及时缴费"}`, ""},
		{"PUT", "/api/v1/user-messages/read-all", `{"customerId":9}`, ""},
		{"GET", "/api/v1/coupons", "", ""},
		{"POST", "/api/v1/coupons", `{"customerId":9,"name":"立减券","amount":1000}`, ""},
		{"PUT", "/api/v1/coupons/CPN-01/disable", "", "CPN-01"},
		{"GET", "/api/v1/invite-config", "", ""},
		{"GET", "/api/v1/user-usages", "", ""},
		{"GET", "/api/v1/diy-guides", "", ""},
		{"PUT", "/api/v1/diy-guides/G-01/toggle", "", "G-01"},
		{"GET", "/api/v1/agreements", "", ""},
		{"PUT", "/api/v1/agreements/AG-01", `{"type":"privacy","version":"v2","content":"..."}`, "AG-01"},
		{"GET", "/api/v1/user-balances", "", ""},
		{"POST", "/api/v1/user-balances/9/adjust", `{"delta":-300}`, ""},
		{"GET", "/api/v1/topup-denominations", "", ""},
		{"PUT", "/api/v1/topup-denominations/D-50", `{"amount":5000,"bonus":200,"active":true}`, "D-50"},
		{"GET", "/api/v1/user-invoices", "", ""},
		{"POST", "/api/v1/user-invoices", `{"customerId":9,"billNo":"BILL-1","invoiceNo":"INV-1","title":"个人"}`, ""},
		{"GET", "/api/v1/user-complaints", "", ""},
		{"POST", "/api/v1/user-complaints/CM-01/close", "", "CM-01"},
		{"GET", "/api/v1/user-verify-records", "", ""},
		{"GET", "/api/v1/product-specs", "", ""},
		{"PUT", "/api/v1/product-specs/P-01", `{"highlights":"千兆","specs":"上下行对等"}`, "P-01"},
		{"GET", "/api/v1/user-bill-items?billNo=BILL-1", "", ""},
	}
	for _, tc := range cases {
		w := doReq(t, r, tc.method, tc.path, tk, tc.body)
		assertOK(t, w)
		if tc.wantCalled != "" && ud.calledPath != tc.wantCalled {
			t.Fatalf("%s %s: called=%q want %q", tc.method, tc.path, ud.calledPath, tc.wantCalled)
		}
	}
}

func TestUserdataDetail_Aggregate(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	ud := &fakeUserdata{row: map[string]any{"customerId": 9, "name": "王先生", "addresses": []map[string]any{}}}
	r := newUserdataRouter(ud, mgr)

	w := getJSON(t, r, "/api/v1/users/9", authToken(t, mgr))
	data := assertOK(t, w)
	if data["name"] != "王先生" {
		t.Fatalf("data=%+v", data)
	}
}

func TestUserdata_NotFound(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	r := newUserdataRouter(&fakeUserdata{row: map[string]any{}}, mgr)

	w := doReq(t, r, http.MethodPut, "/api/v1/addons/ADD-404/toggle", authToken(t, mgr), "")
	var body struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Code == 0 {
		t.Fatalf("toggle 非法 addon 应报错: %s", w.Body.String())
	}
}

var _ userdata.Service = (*fakeUserdata)(nil)
