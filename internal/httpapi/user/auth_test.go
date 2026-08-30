package userapi

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

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// userPortalCustSvc 桩 CustomerService:Get 恒返回主档;List 按 phone 命中。
type userPortalCustSvc struct{ c *customer.Customer }

func (f *userPortalCustSvc) Create(context.Context, customer.Customer) (int64, error) { return 0, nil }
func (f *userPortalCustSvc) Get(context.Context, int64) (*customer.Customer, error)   { return f.c, nil }
func (f *userPortalCustSvc) GetInScope(context.Context, int64, int64, string) (*customer.Customer, error) {
	return f.c, nil
}
func (f *userPortalCustSvc) List(_ context.Context, q customer.CustomerQuery) ([]customer.Customer, error) {
	if q.Phone != "" && q.Phone == f.c.Phone {
		return []customer.Customer{*f.c}, nil
	}
	return nil, nil
}

// userPortalWo 桩 WorkOrderService:CreateComplaint 记录入参,
// ListComplaints / ListComplaintsByCustomerPaged 返回可配置列表;
// GetComplaintByNoAndCustomer 返回按工单号寻址的桩工单。
type userPortalWo struct {
	order.WorkOrderService
	created []order.Complaint
	complts []order.Complaint
}

func (f *userPortalWo) CreateComplaint(_ context.Context, c order.Complaint) (int64, error) {
	f.created = append(f.created, c)
	return int64(len(f.created)), nil
}

func (f *userPortalWo) ListComplaints(context.Context) ([]order.Complaint, error) {
	return f.complts, nil
}

// ListComplaintsByCustomerPaged 桩:不区分 customerID,直接返回预置列表全量。
// hasMore 简单判 len>pageSize;测试用 list 元素少于 pageSize 即一次性返回。
func (f *userPortalWo) ListComplaintsByCustomerPaged(_ context.Context, _ int64, _ int, pageSize int) ([]order.Complaint, bool, error) {
	out := f.complts
	hasMore := false
	if pageSize > 0 && len(out) > pageSize {
		out = out[:pageSize]
		hasMore = true
	}
	return out, hasMore, nil
}

// GetComplaintByNoAndCustomer 桩:按工单号 + customerID(忽略)寻址,未命中返回 ErrOrderNotFound。
func (f *userPortalWo) GetComplaintByNoAndCustomer(_ context.Context, ticketNo string, _ int64) (*order.Complaint, error) {
	for i := range f.complts {
		if f.complts[i].TicketNo == ticketNo {
			return &f.complts[i], nil
		}
	}
	return nil, order.ErrOrderNotFound
}

func newUserPortalRouter(cust *customer.Customer, bills []billing.Bill, byNo *order.Order) (*gin.Engine, *auth.Manager, *userPortalWo) {
	return newUserPortalRouterWithProducts(cust, bills, byNo, nil)
}

// newUserPortalRouterWithProducts 在 newUserPortalRouter 基础上支持注入产品目录列表。
// tests/product detail 需要真实 PUBLISHED 产品列表验证 specs/compare 派生逻辑。
func newUserPortalRouterWithProducts(cust *customer.Customer, bills []billing.Bill, byNo *order.Order, products []customer.ProductOffer) (*gin.Engine, *auth.Manager, *userPortalWo) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("test-secret", time.Hour)
	wo := &userPortalWo{}
	r := gin.New()
	Register(r, &app.Application{
		Customer:         &userPortalCustSvc{c: cust},
		CustomerRealName: &fakeRealName{},
		Billing:          &fakeBilling{bills: bills},
		Order:            &fakeOrder{byNo: byNo},
		Product:          &fakeProduct{list: products},
		WorkOrder:        wo,
		Portal:           portal.NewMemory(),
	}, mgr)
	return r, mgr, wo
}

// (none)

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
	w := userPortalDo(r, http.MethodPost, "/api/user/v1/auth/sms-code",
		`{"phone":"13800001234","scene":"register"}`, "")
	if code, _ := userPortalCode(t, w); code != 0 {
		t.Fatalf("sms-code resp=%s", w.Body.String())
	}
	w = userPortalDo(r, http.MethodPost, "/api/user/v1/auth/register",
		`{"phone":"13800001234","smsCode":"123456","password":"password-10x"}`, "")
	code, data := userPortalCode(t, w)
	if code != 0 || data["customerId"] != float64(7) {
		t.Fatalf("register resp=%s", w.Body.String())
	}
	tok, _ := data["token"].(string)
	if tok == "" {
		t.Fatalf("register no token")
	}
	w = userPortalDo(r, http.MethodGet, "/api/user/v1/profile", "", tok)
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
	if w := userPortalDo(r, http.MethodGet, "/api/user/v1/profile", "", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("no-token code=%d", w.Code)
	}
	adminTok, _ := mgr.Sign(auth.AudAdmin, 1, "admin", "sysadmin")
	w := userPortalDo(r, http.MethodGet, "/api/user/v1/profile", "", adminTok)
	// admin token 在两道防线被拒:Authn aud 不匹配(HTTP 401)或 portalCustomerOnly(envelope 401)。
	if w.Code != http.StatusUnauthorized {
		if code, _ := userPortalCode(t, w); code != int(apitypes.CodeUnauthorized) {
			t.Fatalf("admin token resp=%s", w.Body.String())
		}
	}
}

// TestPortal_BillAndOrderScope 账单可见;他人订单 404(不泄露存在性)。
func TestPortal_BillAndOrderScope(t *testing.T) {
	cust := userPortalCust()
	bills := []billing.Bill{{BillID: 1, BillNo: "B-2026-08-7", CustomerID: 7, Period: "2026-08", Amount: 158, Status: "UNPAID"}}
	ord := &order.Order{ID: 11, OrderNo: "ORD-1", CustomerID: 8, Status: "PENDING"}
	r, mgr, _ := newUserPortalRouter(cust, bills, ord)
	tok, _ := signCustomerToken(mgr, 7, "13800001234")

	w := userPortalDo(r, http.MethodGet, "/api/user/v1/bills/B-2026-08-7", "", tok)
	if code, data := userPortalCode(t, w); code != 0 || data["bill"] == nil {
		t.Fatalf("bill resp=%s", w.Body.String())
	}
	w = userPortalDo(r, http.MethodPost, "/api/user/v1/orders/ORD-1/cancel", "", tok)
	if code, _ := userPortalCode(t, w); code != 40400 {
		t.Fatalf("foreign order resp=%s", w.Body.String())
	}
}

type fakeRealName struct{}

func (f *fakeRealName) ListVerifications(context.Context, int64) ([]customer.RealNameVerification, error) {
	return nil, nil
}
func (f *fakeRealName) AppendVerification(context.Context, customer.RealNameVerification) (int64, error) {
	return 0, nil
}
func (f *fakeRealName) SubmitRealName(context.Context, customer.CustomerRealNameVerification) (int64, error) {
	return 0, nil
}
func (f *fakeRealName) GetLatest(context.Context, int64) (*customer.CustomerRealNameVerification, error) {
	return nil, customer.ErrRealNameNotFound
}
func (f *fakeRealName) Verify(context.Context, int64, string, string, string, int64) error {
	return nil
}

// 三端拆包后与 admin 测试各自持有同形桩(桩实现随端契约独立演进)。
// fakeBilling 桩 billing.BillingService;pays 记录带券落账调用供续费断言。
type fakeBilling struct {
	bills []billing.Bill
	pays  []billing.Payment
}

func (f *fakeBilling) ListBills(context.Context, int64) ([]billing.Bill, error) { return f.bills, nil }
func (f *fakeBilling) CreateBill(context.Context, billing.Bill) (int64, error)  { return 0, nil }
func (f *fakeBilling) GetBill(context.Context, int64) (*billing.Bill, error)    { return nil, nil }
func (f *fakeBilling) ListPayments(context.Context, int64) ([]billing.Payment, error) {
	return nil, nil
}
func (f *fakeBilling) ListPaymentsByCustomer(context.Context, int64) ([]billing.Payment, error) {
	return nil, nil
}
func (f *fakeBilling) CreatePayment(context.Context, billing.Payment) (int64, error) { return 0, nil }
func (f *fakeBilling) PaymentExistsByPayNo(context.Context, string) (bool, error)    { return false, nil }
func (f *fakeBilling) RecordTopup(context.Context, billing.Payment) (int64, error)   { return 0, nil }
func (f *fakeBilling) RecordPayment(context.Context, billing.Payment) (int64, error) { return 0, nil }
func (f *fakeBilling) RecordPaymentWithCoupon(_ context.Context, p billing.Payment) (billing.PaymentReceipt, error) {
	f.pays = append(f.pays, p)
	return billing.PaymentReceipt{PaymentID: 901, Amount: p.Amount}, nil
}
func (f *fakeBilling) GenerateBills(context.Context, string) (int, error) { return 0, nil }
func (f *fakeBilling) RefundPayment(_ context.Context, _ int64, _ string) (*billing.Payment, error) {
	return nil, nil
}
func (f *fakeBilling) DailyCashSummary(context.Context, string) ([]billing.DailyCashRow, error) {
	return nil, nil
}
func (f *fakeBilling) SaveDailyClosing(context.Context, billing.DailyClosing) (billing.DailyClosingResult, error) {
	return billing.DailyClosingResult{Balanced: true}, nil
}
func (f *fakeBilling) CashPaymentsByDate(context.Context, string) ([]billing.Payment, error) {
	return nil, nil
}

// fakeArrears 桩 billing.ArrearsService。
type fakeArrears struct {
	items    []billing.ArrearsItem
	appended *billing.StopResumeTask
	task     *billing.StopResumeTask // GetStopResumeTask 返回
	updated  *struct {               // UpdateStopResumeStatus 落账
		id     int64
		status string
	}
}

// fakeOrder 桩 order.OrderService:订单读/建/跟踪可配置,环节方法返回零值。
type fakeOrder struct {
	list      []order.OrderListItem
	byNo      *order.Order
	byNoErr   error
	trackLog  []order.StageLog
	submitted *order.Order
	submitErr error
}

func (f *fakeOrder) Submit(ctx context.Context, r order.SubmitReq) (*order.Order, error) {
	return f.submitted, f.submitErr
}
func (f *fakeOrder) CheckResource(ctx context.Context, id int64) error  { return nil }
func (f *fakeOrder) Reserve(ctx context.Context, id int64) error        { return nil }
func (f *fakeOrder) ChargeContract(ctx context.Context, id int64) error { return nil }
func (f *fakeOrder) PrepaidAmount(ctx context.Context, id int64) (float64, int, bool, error) {
	return 0, 0, false, nil
}
func (f *fakeOrder) ApplyTag(ctx context.Context, id int64) error { return nil }
func (f *fakeOrder) CreateUserProfile(ctx context.Context, id int64) error {
	return nil
}
func (f *fakeOrder) PreConfigOLT(ctx context.Context, id int64) error  { return nil }
func (f *fakeOrder) DispatchOrder(ctx context.Context, id int64) error { return nil }
func (f *fakeOrder) ScanBind(ctx context.Context, id int64) error      { return nil }
func (f *fakeOrder) ActivateUser(ctx context.Context, id int64) error  { return nil }
func (f *fakeOrder) NotifyActivation(ctx context.Context, id int64) error {
	return nil
}
func (f *fakeOrder) UpdateMap(ctx context.Context, id int64) error { return nil }
func (f *fakeOrder) Cancel(ctx context.Context, id int64) error    { return nil }
func (f *fakeOrder) Release(ctx context.Context, id int64) error   { return nil }
func (f *fakeOrder) RollbackStage(ctx context.Context, id int64) (int8, int8, error) {
	return 0, 0, nil
}
func (f *fakeOrder) List(ctx context.Context, q order.OrderQuery) ([]order.OrderListItem, error) {
	return f.list, nil
}
func (f *fakeOrder) GetByNo(ctx context.Context, no string) (*order.Order, error) {
	return f.byNo, f.byNoErr
}
func (f *fakeOrder) Track(ctx context.Context, id int64) (*order.Order, []order.StageLog, error) {
	return f.byNo, f.trackLog, nil
}

func (f *fakeOrder) ChangeAddress(context.Context, int64, int64) error { return nil }
func (f *fakeOrder) SaveRating(context.Context, order.Rating) error    { return nil }
func (f *fakeOrder) RatingExists(context.Context, string) (bool, error) {
	return false, nil
}

type fakeProduct struct {
	list    []customer.ProductOffer
	changed *customer.ProductOffer // 记录最近一次调价入参
}

func (f *fakeProduct) ListProducts(context.Context, int64) ([]customer.ProductOffer, error) {
	return f.list, nil
}
func (f *fakeProduct) CreateProduct(context.Context, customer.ProductOffer) (int64, error) {
	return 0, nil
}
func (f *fakeProduct) ListRegionOffers(context.Context, int64) ([]customer.RegionOffer, error) {
	return nil, nil
}
func (f *fakeProduct) CreateRegionOffer(context.Context, customer.RegionOffer) (int64, error) {
	return 0, nil
}
func (f *fakeProduct) ChangeProductPrice(_ context.Context, offerID int64, newFee float64, _ time.Time, reason string, _ int64) (int64, error) {
	f.changed = &customer.ProductOffer{ID: offerID, MonthlyFee: newFee}
	return 11, nil
}
func (f *fakeProduct) UpdateProduct(context.Context, int64, string, string, string) error {
	return nil
}
func (f *fakeProduct) UpdateProductStatus(context.Context, int64, string) error {
	return nil
}

// TestPortal_PasswordLogin 注册后账密登录闭环(/auth/login 前缀分离后可用)。
func TestPortal_PasswordLogin(t *testing.T) {
	cust := userPortalCust()
	r, _, _ := newUserPortalRouter(cust, nil, nil)
	// 注册
	if w := userPortalDo(r, http.MethodPost, "/api/user/v1/auth/sms-code",
		`{"phone":"13900005678","scene":"register"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("sms-code: %s", w.Body.String())
	}
	if w := userPortalDo(r, http.MethodPost, "/api/user/v1/auth/register",
		`{"phone":"13900005678","smsCode":"123456","password":"password-10x"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("register: %s", w.Body.String())
	}
	// 账密登录
	w := userPortalDo(r, http.MethodPost, "/api/user/v1/auth/login",
		`{"phone":"13900005678","password":"password-10x"}`, "")
	code, data := userPortalCode(t, w)
	if code != 0 || data["customerId"] == nil {
		t.Fatalf("login resp=%s", w.Body.String())
	}
	// 错密码 401
	w = userPortalDo(r, http.MethodPost, "/api/user/v1/auth/login",
		`{"phone":"13900005678","password":"wrong-pass-99"}`, "")
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeUnauthorized) {
		t.Fatalf("bad password resp=%s", w.Body.String())
	}
}

// TestPortal_VerifySubmitFlow 实名分步流程:发码到绑定手机 → 提交(验证码+证件附件) → 状态含脱敏手机号。
// memory 门户发码恒为 123456;错码不落库不消费,正码可复用重发后的新码。
func TestPortal_VerifySubmitFlow(t *testing.T) {
	cust := userPortalCust()
	r, mgr, _ := newUserPortalRouter(cust, nil, nil)
	tok, _ := signCustomerToken(mgr, 7, "13800001234")

	w := userPortalDo(r, http.MethodPost, "/api/user/v1/auth/verify/sms-code", ``, tok)
	if code, data := userPortalCode(t, w); code != 0 || data["phoneMasked"] != "138****1234" {
		t.Fatalf("sms-code resp=%s", w.Body.String())
	}
	// 缺验证码/缺证件附件 → 参数错误
	w = userPortalDo(r, http.MethodPost, "/api/user/v1/auth/verify",
		`{"name":"王先生","idNo":"110101199001011234"}`, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeInvalidParam) {
		t.Fatalf("missing fields resp=%s", w.Body.String())
	}
	// 错验证码 → 40100
	w = userPortalDo(r, http.MethodPost, "/api/user/v1/auth/verify",
		`{"name":"王先生","idNo":"110101199001011234","smsCode":"000000","idCardFrontId":1,"idCardBackId":2}`, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeUnauthorized) {
		t.Fatalf("bad code resp=%s", w.Body.String())
	}
	// 正码提交成功
	w = userPortalDo(r, http.MethodPost, "/api/user/v1/auth/verify",
		`{"name":"王先生","idNo":"110101199001011234","smsCode":"123456","idCardFrontId":1,"idCardBackId":2}`, tok)
	if code, data := userPortalCode(t, w); code != 0 || data["ok"] != true {
		t.Fatalf("submit resp=%s", w.Body.String())
	}
	// 状态查询带脱敏手机号与最新结论字段
	w = userPortalDo(r, http.MethodGet, "/api/user/v1/auth/verify", "", tok)
	if code, data := userPortalCode(t, w); code != 0 || data["phoneMasked"] != "138****1234" ||
		data["latestResult"] == nil || data["rejectReason"] == nil {
		t.Fatalf("status resp=%s", w.Body.String())
	}
}

// TestPortal_ProductDetail 套餐详情:真实 PUBLISHED 产品必须返回非空 specs/description/compare,
// 不再依赖前端文案占位——后端从 ProductOffer 派生,客户端契约字段齐备即可直接渲染。
func TestPortal_ProductDetail(t *testing.T) {
	cust := userPortalCust()
	r, mgr, _ := newUserPortalRouterWithProducts(cust, nil, nil, []customer.ProductOffer{
		{ID: 1, LegalEntityID: 1, Name: "300M 畅享宽带", Bandwidth: "300M", MonthlyFee: 99.00, Category: "broadband", Status: "PUBLISHED"},
		{ID: 2, LegalEntityID: 1, Name: "500M 极速宽带", Bandwidth: "500M", MonthlyFee: 129.00, Category: "broadband", Status: "PUBLISHED"},
		{ID: 3, LegalEntityID: 1, Name: "200M 入门宽带", Bandwidth: "200M", MonthlyFee: 79.00, Category: "broadband", Status: "PUBLISHED"},
		{ID: 4, LegalEntityID: 1, Name: "5G 融合套餐", Bandwidth: "1Gbps", MonthlyFee: 199.00, Category: "fusion", Status: "PUBLISHED"},
		{ID: 5, LegalEntityID: 1, Name: "草稿产品", Bandwidth: "1000M", MonthlyFee: 299.00, Category: "broadband", Status: "DRAFT"},
	})
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)

	// 命中 PUBLISHED:6 项 specs 全有值、description 非空、compare 含同 category 候选且排除自己。
	w := userPortalDo(r, http.MethodGet, "/api/user/v1/products/1", "", tok)
	code, data := userPortalCode(t, w)
	if code != 0 {
		t.Fatalf("code=%d resp=%s", code, w.Body.String())
	}
	specs, _ := data["specs"].([]any)
	if len(specs) != 5 {
		t.Fatalf("specs len=%d want 5", len(specs))
	}
	for i, s := range specs {
		m := s.(map[string]any)
		if m["label"] == nil || m["value"] == nil {
			t.Fatalf("spec[%d] missing label/value", i)
		}
	}
	prod := data["product"].(map[string]any)
	if prod["description"] == "" {
		t.Fatalf("product.description empty, want derived text")
	}
	if prod["contractMonths"].(float64) != 12 {
		t.Fatalf("contractMonths=%v want 12 for broadband", prod["contractMonths"])
	}
	if prod["featured"] != true {
		t.Fatalf("featured=%v want true for fee=99", prod["featured"])
	}
	compare, _ := data["compare"].([]any)
	if len(compare) != 2 {
		t.Fatalf("compare len=%d want 2 (excludes self id=1 + excludes fusion)", len(compare))
	}
	for _, c := range compare {
		cm := c.(map[string]any)
		if cm["productId"] == "1" {
			t.Fatalf("compare contains self id=1")
		}
		if cm["category"] != "broadband" {
			t.Fatalf("compare contains non-broadband: %v", cm)
		}
	}

	// 草稿产品 404:detail 只暴露 PUBLISHED,草稿不可见。
	w = userPortalDo(r, http.MethodGet, "/api/user/v1/products/5", "", tok)
	code, _ = userPortalCode(t, w)
	if code != int(apitypes.CodeNotFound) {
		t.Fatalf("draft code=%d resp=%s", code, w.Body.String())
	}
}
