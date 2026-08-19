package app

// 用户端门户最小单测:注册发码流程 + 客户鉴权 + 按客户过滤(账单/订单归属)。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// userPortalCustSvc 桩 CustomerService:Get 恒返回主档;List 按 phone 命中。
type userPortalCustSvc struct{ c *customer.Customer }

func (f *userPortalCustSvc) Create(context.Context, customer.Customer) (int64, error) { return 0, nil }
func (f *userPortalCustSvc) Get(context.Context, int64) (*customer.Customer, error)   { return f.c, nil }
func (f *userPortalCustSvc) List(_ context.Context, q customer.CustomerQuery) ([]customer.Customer, error) {
	if q.Phone != "" && q.Phone == f.c.Phone {
		return []customer.Customer{*f.c}, nil
	}
	return nil, nil
}

// userPortalWo 桩 WorkOrderService:CreateComplaint 记录入参。
type userPortalWo struct {
	order.WorkOrderService
	created []order.Complaint
}

func (f *userPortalWo) CreateComplaint(_ context.Context, c order.Complaint) (int64, error) {
	f.created = append(f.created, c)
	return int64(len(f.created)), nil
}

func newUserPortalRouter(cust *customer.Customer, bills []billing.Bill, byNo *order.Order) (*gin.Engine, *auth.Manager, *userPortalWo) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("test-secret", time.Hour)
	wo := &userPortalWo{}
	r := gin.New()
	registerUserPortalRoutes(r, &Application{
		Customer:         &userPortalCustSvc{c: cust},
		CustomerRealName: &fakeRealName{},
		Billing:          &fakeBilling{bills: bills},
		Order:            &fakeOrder{byNo: byNo},
		Product:          &fakeProduct{},
		WorkOrder:        wo,
	}, mgr)
	return r, mgr, wo
}

func userPortalCust() *customer.Customer {
	return &customer.Customer{ID: 7, Name: "王先生", Phone: "13800001234",
		IdType: "身份证", IdNo: "110101199001011234", RealNameStatus: "PENDING", ServiceStatus: "ACTIVE"}
}

// portalDo 门户测试公共助手:发请求并解析 envelope(code/msg/data)+_status(HTTP 状态)。
// 供用户端门户与师傅端门户测试共用。
func portalDo(r *gin.Engine, method, path, body, token string) map[string]any {
	w := userPortalDo(r, method, path, body, token)
	out := map[string]any{"_status": w.Code}
	if w.Body.Len() == 0 {
		return out
	}
	var env map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &env); err == nil {
		for k, v := range env {
			out[k] = v
		}
	}
	return out
}

func userPortalDo(r *gin.Engine, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func userPortalCode(t *testing.T, w *httptest.ResponseRecorder) (int, map[string]any) {
	t.Helper()
	var resp struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return resp.Code, resp.Data
}

// TestPortal_RegisterAndProfile 发码 → 注册拿 token → 带token取我的档案。
func TestPortal_RegisterAndProfile(t *testing.T) {
	cust := userPortalCust()
	r, mgr, _ := newUserPortalRouter(cust, nil, nil)
	portalSmsCodeGen = func() string { return "123456" }
	defer func() { portalSmsCodeGen = nil }()

	w := userPortalDo(r, http.MethodPost, "/api/v1/auth/sms-code",
		`{"phone":"13800001234","scene":"register"}`, "")
	if code, _ := userPortalCode(t, w); code != 0 {
		t.Fatalf("sms-code resp=%s", w.Body.String())
	}
	w = userPortalDo(r, http.MethodPost, "/api/v1/auth/register",
		`{"phone":"13800001234","smsCode":"123456","password":"password-10x"}`, "")
	code, data := userPortalCode(t, w)
	if code != 0 || data["customerId"] != float64(7) {
		t.Fatalf("register resp=%s", w.Body.String())
	}
	tok, _ := data["token"].(string)
	if tok == "" {
		t.Fatalf("register no token")
	}
	w = userPortalDo(r, http.MethodGet, "/api/v1/profile", "", tok)
	if code, data := userPortalCode(t, w); code != 0 || data["name"] != "王先生" {
		t.Fatalf("profile resp=%s", w.Body.String())
	}
	// 同密钥客户 token 可被 Manager 校验;身份编码回读一致。
	claims, err := mgr.Verify(tok)
	if err != nil || customerIDFromToken(claims) != 7 {
		t.Fatalf("claims=%+v err=%v", claims, err)
	}
}

// TestPortal_Unauthorized 无 token / 管理员 token 访问客户端点均 401。
func TestPortal_Unauthorized(t *testing.T) {
	cust := userPortalCust()
	r, mgr, _ := newUserPortalRouter(cust, nil, nil)
	if w := userPortalDo(r, http.MethodGet, "/api/v1/profile", "", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("no-token code=%d", w.Code)
	}
	adminTok, _ := mgr.Sign(1, "admin", "sysadmin")
	w := userPortalDo(r, http.MethodGet, "/api/v1/profile", "", adminTok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeUnauthorized) {
		t.Fatalf("admin token resp=%s", w.Body.String())
	}
}

// TestPortal_BillAndOrderScope 账单可见;他人订单 404(不泄露存在性)。
func TestPortal_BillAndOrderScope(t *testing.T) {
	cust := userPortalCust()
	bills := []billing.Bill{{BillID: 1, BillNo: "B-2026-08-7", CustomerID: 7, Period: "2026-08", Amount: 158, Status: "UNPAID"}}
	ord := &order.Order{ID: 11, OrderNo: "ORD-1", CustomerID: 8, Status: "PENDING"}
	r, mgr, _ := newUserPortalRouter(cust, bills, ord)
	tok, _ := signCustomerToken(mgr, 7, "13800001234")

	w := userPortalDo(r, http.MethodGet, "/api/v1/bills/B-2026-08-7", "", tok)
	if code, data := userPortalCode(t, w); code != 0 || data["bill"] == nil {
		t.Fatalf("bill resp=%s", w.Body.String())
	}
	w = userPortalDo(r, http.MethodPost, "/api/v1/orders/ORD-1/cancel", "", tok)
	if code, _ := userPortalCode(t, w); code != 40400 {
		t.Fatalf("foreign order resp=%s", w.Body.String())
	}
}
