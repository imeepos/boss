package userapi

// 用户端门户缺失端点单测:产品/增值/订单/账单/缴费/优惠券/地址/投诉/登出。
// 契约:api/openapi/user/*.yaml;全部走 JWT 客户鉴权(auth.AudUser)。

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/customer"
	udcustomer "github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// fakeUserData 桩 userdata.Service:内存 map 满足读端点。
type fakeUserData struct {
	addrs   []map[string]any
	coupons []map[string]any
	addons  []map[string]any
	subs    []map[string]any
	plans   []map[string]any
	invite  []map[string]any
	created int
	// 续费桩:renewFee 0 = 套餐未命中;renewedMonths/renewEnd 供断言。
	renewProductID int64
	renewFee       int
	renewedMonths  int
	renewEnd       string
}

func (f *fakeUserData) ListUsers(context.Context, string) ([]map[string]any, error) {
	return nil, nil
}
func (f *fakeUserData) GetUserDetail(context.Context, int64) (map[string]any, error) {
	return nil, nil
}
func (f *fakeUserData) UpdateUserAccount(context.Context, int64, udcustomer.UserAccountUpdate) error {
	return nil
}
func (f *fakeUserData) ListUserAccounts(context.Context) ([]map[string]any, error) {
	return nil, nil
}
func (f *fakeUserData) ListUserAddresses(context.Context) ([]map[string]any, error) {
	return f.addrs, nil
}
func (f *fakeUserData) CreateUserAddress(context.Context, udcustomer.UserAddress) (int64, error) {
	f.created++
	return int64(f.created), nil
}
func (f *fakeUserData) UpdateUserAddress(context.Context, int64, int64, udcustomer.UserAddress) error {
	return nil
}
func (f *fakeUserData) DeleteUserAddress(context.Context, int64, int64) error {
	return nil
}
func (f *fakeUserData) ListUserPlans(context.Context) ([]map[string]any, error) {
	return f.plans, nil
}
func (f *fakeUserData) CreateUserPlan(context.Context, udcustomer.UserPlan) (int64, error) {
	return 0, nil
}
func (f *fakeUserData) GetPlanForRenewal(_ context.Context, _, _ int64) (int64, float64, error) {
	if f.renewFee == 0 {
		return 0, 0, udcustomer.ErrPlanNotFound
	}
	return f.renewProductID, float64(f.renewFee), nil
}
func (f *fakeUserData) RenewPlan(_ context.Context, _, _ int64, months int) (string, error) {
	f.renewedMonths = months
	return f.renewEnd, nil
}
func (f *fakeUserData) ListAddons(context.Context) ([]map[string]any, error) { return f.addons, nil }
func (f *fakeUserData) CreateAddon(context.Context, udcustomer.Addon) error  { return nil }
func (f *fakeUserData) ToggleAddon(context.Context, string) error            { return nil }
func (f *fakeUserData) ListAddonSubscriptions(context.Context) ([]map[string]any, error) {
	return f.subs, nil
}
func (f *fakeUserData) CreateAddonSubscription(context.Context, udcustomer.AddonSubscription) (int64, error) {
	f.created++
	return int64(f.created), nil
}
func (f *fakeUserData) ListNotifySettings(context.Context) ([]map[string]any, error) {
	return nil, nil
}
func (f *fakeUserData) UpdateNotifySettings(context.Context, int64, udcustomer.NotifySetting) error {
	return nil
}
func (f *fakeUserData) ListUserFaqs(context.Context) ([]map[string]any, error) { return nil, nil }
func (f *fakeUserData) CreateUserFaq(context.Context, udcustomer.UserFaq) error {
	return nil
}
func (f *fakeUserData) ToggleUserFaq(context.Context, string) error { return nil }
func (f *fakeUserData) ListUserMessages(context.Context, string) ([]map[string]any, error) {
	return nil, nil
}
func (f *fakeUserData) CreateUserMessage(context.Context, udcustomer.UserMessage) (int64, error) {
	return 0, nil
}
func (f *fakeUserData) MarkAllMessagesRead(context.Context, int64) error { return nil }
func (f *fakeUserData) ListCoupons(context.Context) ([]map[string]any, error) {
	return f.coupons, nil
}
func (f *fakeUserData) CreateCoupon(context.Context, udcustomer.Coupon) error { return nil }
func (f *fakeUserData) DisableCoupon(context.Context, string) error           { return nil }
func (f *fakeUserData) GetInviteConfig(context.Context) ([]map[string]any, error) {
	return f.invite, nil
}
func (f *fakeUserData) UpdateInviteConfig(context.Context, string, int64) error { return nil }
func (f *fakeUserData) ListUserUsages(context.Context) ([]map[string]any, error) {
	return nil, nil
}
func (f *fakeUserData) ListDiyGuides(context.Context) ([]map[string]any, error) { return nil, nil }
func (f *fakeUserData) ToggleDiyGuide(context.Context, string) error            { return nil }
func (f *fakeUserData) ListAgreements(context.Context) ([]map[string]any, error) {
	return nil, nil
}
func (f *fakeUserData) UpdateAgreement(context.Context, string, udcustomer.Agreement) error {
	return nil
}
func (f *fakeUserData) ListUserBalances(context.Context) ([]map[string]any, error) {
	return nil, nil
}
func (f *fakeUserData) AdjustUserBalance(context.Context, int64, int64) error { return nil }
func (f *fakeUserData) ListTopupDenominations(context.Context) ([]map[string]any, error) {
	return nil, nil
}
func (f *fakeUserData) UpdateTopupDenomination(context.Context, string, udcustomer.TopupDenomination) error {
	return nil
}
func (f *fakeUserData) ListUserInvoices(context.Context) ([]map[string]any, error) { return nil, nil }
func (f *fakeUserData) CreateUserInvoice(context.Context, udcustomer.UserInvoice) (int64, error) {
	return 0, nil
}
func (f *fakeUserData) ListUserComplaints(context.Context) ([]map[string]any, error) {
	return nil, nil
}
func (f *fakeUserData) CloseUserComplaint(context.Context, string) error { return nil }
func (f *fakeUserData) ListUserVerifyRecords(context.Context) ([]map[string]any, error) {
	return nil, nil
}
func (f *fakeUserData) ListProductSpecs(context.Context) ([]map[string]any, error) {
	return nil, nil
}
func (f *fakeUserData) UpdateProductSpec(context.Context, string, udcustomer.ProductSpec) error {
	return nil
}
func (f *fakeUserData) ListUserBillItems(context.Context, string) ([]map[string]any, error) {
	return nil, nil
}

// fakeChannelStub 桩 order.ChannelService。
type fakeChannelStub struct{ chans []order.Channel }

func (f *fakeChannelStub) ListChannels(context.Context) ([]order.Channel, error) {
	return f.chans, nil
}
func (f *fakeChannelStub) GetChannel(context.Context, int64) (*order.Channel, error) {
	return nil, nil
}
func (f *fakeChannelStub) CreateChannel(context.Context, order.Channel) (int64, error) { return 0, nil }

// fakeTaxStub 桩 billing.TaxService。
type fakeTaxStub struct {
	invs      []billing.Invoice
	issueResp *billing.Invoice
	issueErr  error
	issuedNo  []string
}

func (f *fakeTaxStub) ListInvoices(context.Context, int64) ([]billing.Invoice, error) {
	return f.invs, nil
}
func (f *fakeTaxStub) IssueInvoicesForPeriod(context.Context, string) (billing.InvoiceRunResult, error) {
	return billing.InvoiceRunResult{}, nil
}
func (f *fakeTaxStub) IssueInvoiceForBill(_ context.Context, _ int64, billNo string) (*billing.Invoice, error) {
	f.issuedNo = append(f.issuedNo, billNo)
	if f.issueErr != nil {
		return nil, f.issueErr
	}
	if f.issueResp == nil {
		f.issueResp = &billing.Invoice{ID: 11, InvoiceNo: "INV-00000001"}
	}
	return f.issueResp, nil
}
func (f *fakeTaxStub) VoidInvoice(context.Context, int64, string) error { return nil }
func (f *fakeTaxStub) ReissueInvoice(context.Context, int64) (*billing.Invoice, error) {
	return nil, nil
}
func (f *fakeTaxStub) GetInvoice(context.Context, int64) (*billing.Invoice, error) {
	return nil, nil
}
func (f *fakeTaxStub) BackfillTaxNo(context.Context, int64, string) error { return nil }
func (f *fakeTaxStub) MarkTaxResult(context.Context, int64, billing.TaxReceipt) error {
	return nil
}

// newFullPortalRouter 装配全部门户域(含 UserData/Channel/Tax),供新端点单测。
func newFullPortalRouter(cust *customer.Customer, ud *fakeUserData, ch *fakeChannelStub,
	bills []billing.Bill, prods []customer.ProductOffer, list []order.OrderListItem,
	byNo *order.Order) (*gin.Engine, *auth.Manager, *userPortalWo) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("test-secret", time.Hour)
	wo := &userPortalWo{}
	r := gin.New()
	Register(r, &app.Application{
		Customer:  &userPortalCustSvc{c: cust},
		Billing:   &fakeBilling{bills: bills},
		Order:     &fakeOrder{list: list, byNo: byNo},
		Product:   &fakeProduct{list: prods},
		WorkOrder: wo,
		Portal:    portal.NewMemory(),
		UserData:  ud,
		Promotion: &fakePromo{items: ud.coupons},
		Channel:   ch,
		Tax:       &fakeTaxStub{},
	}, mgr)
	return r, mgr, wo
}

// TestPortal_ProductsAndAddons 产品列表 + 增值服务[(不依赖 UserData 时的空兜底)]。
func TestPortal_ProductsAndAddons(t *testing.T) {
	cust := userPortalCust()
	ud := &fakeUserData{
		addons: []map[string]any{
			{"addonId": "iptv", "name": "IPTV", "price": int64(1000), "status": "on"},
			{"addonId": "wifi6", "name": "WiFi6", "price": int64(500), "status": "on"},
		},
		subs: []map[string]any{
			{"customerId": int64(7), "addonId": "iptv", "action": "subscribe"},
		},
	}
	prods := []customer.ProductOffer{{ID: 101, Name: "家庭宽带100M", MonthlyFee: 99, Status: "PUBLISHED"}}
	r, mgr, _ := newFullPortalRouter(cust, ud, nil, nil, prods, nil, nil)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)

	w := userPortalDo(r, http.MethodGet, "/api/user/v1/products", ``, tok)
	code, data := userPortalCode(t, w)
	if code != int(apitypes.CodeOK) {
		t.Fatalf("products resp=%s", w.Body.String())
	}
	items, _ := data["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("products items=%v", data["items"])
	}

	w = userPortalDo(r, http.MethodGet, "/api/user/v1/addons", ``, tok)
	code, data = userPortalCode(t, w)
	if code != int(apitypes.CodeOK) {
		t.Fatalf("addons resp=%s", w.Body.String())
	}
	avail, _ := data["available"].([]any)
	owned, _ := data["subscribed"].([]any)
	if len(avail) != 1 || len(owned) != 1 {
		t.Fatalf("addons avail=%v owned=%v", avail, owned)
	}
}

// TestPortal_BillsPayments:账单列表 + 缴费 + 流水 + 发票聚合。
func TestPortal_BillsPayments(t *testing.T) {
	cust := userPortalCust()
	bills := []billing.Bill{{BillID: 1, BillNo: "B-001", CustomerID: 7, Period: "2026-08", Amount: 99, Status: "UNPAID"}}
	r, mgr, _ := newFullPortalRouter(cust, &fakeUserData{plans: []map[string]any{
		{"customerId": int64(7), "planName": "家庭宽带100M"},
	}}, nil, bills, nil, nil, nil)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)

	w := userPortalDo(r, http.MethodGet, "/api/user/v1/bills", ``, tok)
	if code, data := userPortalCode(t, w); code != int(apitypes.CodeOK) {
		t.Fatalf("bills resp=%s", w.Body.String())
	} else if items, _ := data["items"].([]any); len(items) != 1 {
		t.Fatalf("bills items=%v", data["items"])
	}

	w = userPortalDo(r, http.MethodPost, "/api/user/v1/payments",
		`{"billNo":"B-001","amount":99,"payMethod":"wechat"}`, tok)
	if code, data := userPortalCode(t, w); code != int(apitypes.CodeOK) {
		t.Fatalf("pay resp=%s", w.Body.String())
	} else if data["payNo"] == nil {
		t.Fatalf("pay no payNo: %v", data)
	}

	w = userPortalDo(r, http.MethodGet, "/api/user/v1/invoices", ``, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeOK) {
		t.Fatalf("invoices resp=%s", w.Body.String())
	}
}

// TestPortal_CouponsAddresses:优惠券 + 地址列表 + 新增地址落库。
func TestPortal_CouponsAddresses(t *testing.T) {
	cust := userPortalCust()
	ud := &fakeUserData{
		coupons: []map[string]any{
			{"couponId": "C-001", "customerId": int64(7), "name": "满100减20", "amount": int64(2000), "status": "active"},
		},
		invite: []map[string]any{{"inviteLink": "https://u.example/invite"}},
		addrs: []map[string]any{
			{"id": int64(1), "customerId": int64(7), "detail": "Sunrise B8-502", "contact": "王先生", "phone": "13800001234", "isDefault": true},
		},
	}
	r, mgr, _ := newFullPortalRouter(cust, ud, nil, nil, nil, nil, nil)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)

	w := userPortalDo(r, http.MethodGet, "/api/user/v1/coupons?status=available", ``, tok)
	if code, data := userPortalCode(t, w); code != int(apitypes.CodeOK) {
		t.Fatalf("coupons resp=%s", w.Body.String())
	} else if items, _ := data["items"].([]any); len(items) != 1 {
		t.Fatalf("coupons items=%v", data["items"])
	}

	w = userPortalDo(r, http.MethodGet, "/api/user/v1/addresses", ``, tok)
	if code, data := userPortalCode(t, w); code != int(apitypes.CodeOK) {
		t.Fatalf("addresses resp=%s", w.Body.String())
	} else if items, _ := data["items"].([]any); len(items) != 1 {
		t.Fatalf("addresses items=%v", data["items"])
	}

	w = userPortalDo(r, http.MethodPost, "/api/user/v1/addresses",
		`{"community":"Sunrise","building":"8","door":"502","contact":"王先生","phone":"13800001234"}`, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeOK) {
		t.Fatalf("create address resp=%s", w.Body.String())
	}
	if ud.created != 1 {
		t.Fatalf("expected address persisted, created=%d", ud.created)
	}
}

// TestPortal_OrderList:按客户过滤的订单列表 + 下单。
func TestPortal_OrderList(t *testing.T) {
	cust := userPortalCust()
	list := []order.OrderListItem{{OrderNo: "ORD-20260819-000311", Product: "家庭宽带100M",
		Stage: 12, Status: "DONE", AddressID: 1, Address: "Sunrise B8-502"}}
	r, mgr, _ := newFullPortalRouter(cust, &fakeUserData{}, &fakeChannelStub{chans: []order.Channel{{ID: 2, Code: "ONLINE", Status: "ACTIVE"}}},
		nil, nil, list, &order.Order{ID: 9, OrderNo: "ORD-20260819-000311", CustomerID: 7, OfferID: 101, AddressID: 1, Stage: 12, Status: "DONE"})
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)

	w := userPortalDo(r, http.MethodGet, "/api/user/v1/orders?status=done", ``, tok)
	if code, data := userPortalCode(t, w); code != int(apitypes.CodeOK) {
		t.Fatalf("orders resp=%s", w.Body.String())
	} else if items, _ := data["items"].([]any); len(items) != 1 {
		t.Fatalf("orders items=%v", data["items"])
	}

	w = userPortalDo(r, http.MethodGet, "/api/user/v1/orders/ORD-20260819-000311", ``, tok)
	if code, data := userPortalCode(t, w); code != int(apitypes.CodeOK) {
		t.Fatalf("order detail resp=%s", w.Body.String())
	} else if data["order"] == nil {
		t.Fatalf("order detail missing order: %v", data)
	}
}

// TestPortal_Complaints:投诉列表(按客户过滤)。
func TestPortal_Complaints(t *testing.T) {
	cust := userPortalCust()
	r, mgr, wo := newFullPortalRouter(cust, &fakeUserData{}, nil, nil, nil, nil, nil)
	wo.complts = []order.Complaint{{TicketNo: "TKT-001", CustomerID: 7, Type: "quality", Status: "OPEN"}}
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)
	w := userPortalDo(r, http.MethodGet, "/api/user/v1/complaints", ``, tok)
	if code, data := userPortalCode(t, w); code != int(apitypes.CodeOK) {
		t.Fatalf("complaints resp=%s", w.Body.String())
	} else if items, _ := data["items"].([]any); len(items) != 1 {
		t.Fatalf("complaints items=%v", data["items"])
	}
}

// TestPortal_Logout:登出(无状态 JWT,返回 ok 即可)。
func TestPortal_Logout(t *testing.T) {
	cust := userPortalCust()
	r, mgr, _ := newFullPortalRouter(cust, &fakeUserData{}, nil, nil, nil, nil, nil)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)
	w := userPortalDo(r, http.MethodPost, "/api/user/v1/auth/logout", ``, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeOK) {
		t.Fatalf("logout resp=%s", w.Body.String())
	}
}
