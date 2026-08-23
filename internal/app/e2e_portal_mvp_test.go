package app_test

// PORT 客户门户 MVP 验收集(2028 Q2 交付,真实 PG):单一客户会话走完
// 浏览 → 订购 → 订单进度 → 账单 → 券抵扣缴费 → 自动积分/等级 → 发票 → 工单 → 券仓/积分流水。
// 事实源约束:订单/账务走 a.Order/a.Billing 真实链路,门户 HTTP 不绕过。
// 运行: BOSS_PG_TEST_DSN="host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable" \
//   go test ./internal/app/ -run TestE2E_PortalMVP -v -count=1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/promotion"
	"github.com/ymm-001/boss/internal/httpapi"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/config"
)

const portalMvpPassword = "Portal-mvp-123"

// portalMvpJSON 通用请求:带客户 token 的 JSON 调用,返回 data 节点。
func portalMvpJSON(t *testing.T, ts *httptest.Server, method, path, body, token string) map[string]any {
	t.Helper()
	var rd *strings.Reader
	if body == "" {
		rd = strings.NewReader("{}")
	} else {
		rd = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, ts.URL+path, rd)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	var envelope struct {
		Code int            `json:"code"`
		Msg  string         `json:"msg"`
		Data map[string]any `json:"data"`
	}
	raw := make([]byte, 1<<20)
	n, _ := resp.Body.Read(raw)
	if err := json.Unmarshal(raw[:n], &envelope); err != nil {
		t.Fatalf("%s %s: bad json %s", method, path, string(raw[:n]))
	}
	if resp.StatusCode != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("%s %s: http=%d code=%d msg=%s body=%s", method, path, resp.StatusCode, envelope.Code, envelope.Msg, string(raw[:n]))
	}
	return envelope.Data
}

func portalMvpArr(t *testing.T, data map[string]any, key string) []any {
	t.Helper()
	v, ok := data[key].([]any)
	if !ok {
		t.Fatalf("data[%s] 不是数组: %#v", key, data[key])
	}
	return v
}

// TestE2E_PortalMVP_Integration 客户门户 MVP 全旅程。
func TestE2E_PortalMVP_Integration(t *testing.T) {
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置,跳过集成测试")
	}
	ctx := context.Background()
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	cfg.Database.DSN = dsn
	a, err := app.New(ctx, cfg, "../../migrations")
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	defer a.Close()

	pool := portalMvpPool(t, dsn)
	s := seedE2E(t, ctx, pool, a)

	var phone string
	if err := pool.QueryRow(ctx,
		`SELECT phone FROM customers WHERE id=$1`, s.customerID).Scan(&phone); err != nil {
		t.Fatal(err)
	}
	portalMvpCleanup(t, ctx, pool, phone, s.customerID)

	// 门户账号 + 登录。
	if _, err := a.Portal.UpsertAccount(ctx, phone, portalMvpPassword, s.customerID); err != nil {
		t.Fatalf("portal account: %v", err)
	}
	r := gin.New()
	httpapi.RegisterRoutes(r, a, auth.NewManager("portal-mvp-secret", time.Hour))
	ts := httptest.NewServer(r)
	defer ts.Close()

	login := portalMvpJSON(t, ts, http.MethodPost, "/api/user/v1/auth/login",
		fmt.Sprintf(`{"phone":%q,"mode":"password","password":%q}`, phone, portalMvpPassword), "")
	tok, _ := login["token"].(string)
	if tok == "" {
		t.Fatalf("login 无 token: %#v", login)
	}

	// ---- 1. 产品浏览 ----
	prods := portalMvpJSON(t, ts, http.MethodGet, "/api/user/v1/products", "", tok)
	found := false
	for _, it := range portalMvpArr(t, prods, "items") {
		m, _ := it.(map[string]any)
		if name, _ := m["name"].(string); strings.HasPrefix(name, "E2E套餐") {
			if id, _ := m["productId"].(string); id == fmt.Sprintf("%d", s.offerID) {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("产品目录未见种子套餐 offer=%d: %#v", s.offerID, prods)
	}

	// ---- 2. 订购(后付费,环节推进后由账单缴费) ----
	addr := newE2EAddress(t, ctx, pool, a, "e2e")
	ord := portalMvpJSON(t, ts, http.MethodPost, "/api/user/v1/orders",
		fmt.Sprintf(`{"productId":"%d","addressId":"%d","channelId":"%d","billingMode":"POSTPAID"}`,
			s.offerID, addr, s.channelID), tok)
	orderNo, _ := ord["orderNo"].(string)
	if orderNo == "" {
		t.Fatalf("下单无 orderNo: %#v", ord)
	}

	// ---- 3. 订单进度(门户视角可查) ----
	detail := portalMvpJSON(t, ts, http.MethodGet, "/api/user/v1/orders/"+orderNo, "", tok)
	ordDetail, _ := detail["order"].(map[string]any)
	if ordDetail == nil {
		t.Fatalf("订单详情缺 order 节点: %#v", detail)
	}
	if v, _ := ordDetail["orderNo"].(string); v != orderNo {
		t.Fatalf("订单详情 orderNo=%v", v)
	}
	// 后台推进到环节4(真实链路,不绕过状态机)。
	var orderID int64
	if err := pool.QueryRow(ctx,
		`SELECT id FROM orders WHERE order_no=$1`, orderNo).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	for _, step := range []func(context.Context, int64) error{
		a.Order.CheckResource, a.Order.Reserve,
	} {
		if err := step(ctx, orderID); err != nil {
			t.Fatalf("环节推进: %v", err)
		}
	}

	// ---- 4. 账单(月账生成后门户可见) ----
	billID, err := a.Billing.CreateBill(ctx, billing.Bill{
		BillNo: "BILL-MVP-" + orderNo, CustomerID: s.customerID,
		CustomerName: "E2E客户", LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		Period: time.Now().Format("2006-01"), Amount: 99.0, Status: "UNPAID",
	})
	if err != nil {
		t.Fatalf("CreateBill: %v", err)
	}
	_ = billID
	bills := portalMvpJSON(t, ts, http.MethodGet, "/api/user/v1/bills", "", tok)
	var billNo string
	for _, it := range portalMvpArr(t, bills, "items") {
		m, _ := it.(map[string]any)
		if no, _ := m["billNo"].(string); no == "BILL-MVP-"+orderNo {
			billNo = no
		}
	}
	if billNo == "" {
		t.Fatalf("门户账单未见种子账单: %#v", bills)
	}

	// ---- 5. 券抵扣缴费 + 自动积分 ----
	tplID, err := a.Promotion.CreateTemplate(ctx, promotion.Template{
		LegalEntityID: 1, Name: "MVP验收券-" + orderNo, Type: promotion.TypeCash,
		FaceValue: 500, ValidDays: 30,
	})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	couponID, err := a.Promotion.IssueToCustomer(ctx, tplID, s.customerID, promotion.SourceAdmin)
	if err != nil {
		t.Fatalf("IssueToCustomer: %v", err)
	}
	if _, err := a.Points.SaveEarnRule(ctx, promotion2EarnRule()); err != nil {
		t.Fatalf("SaveEarnRule: %v", err)
	}
	pay := portalMvpJSON(t, ts, http.MethodPost, "/api/user/v1/payments",
		fmt.Sprintf(`{"billNo":%q,"amount":99,"payMethod":"cash","couponId":%q}`, billNo, couponID), tok)
	if d, _ := pay["deductedCents"].(float64); int64(d) != 500 {
		t.Fatalf("券抵扣金额=%v want 500: %#v", pay["deductedCents"], pay)
	}

	// ---- 6. 积分与等级(缴费自动积分) ----
	pts := portalMvpJSON(t, ts, http.MethodGet, "/api/user/v1/points", "", tok)
	if bal, _ := pts["balance"].(float64); bal != 99 { // 99 元 × 1 分/元
		t.Fatalf("积分余额=%v want 99: %#v", pts["balance"], pts)
	}
	sawEarn := false
	for _, it := range portalMvpArr(t, pts, "entries") {
		m, _ := it.(map[string]any)
		if r, _ := m["reason"].(string); r == "PAYMENT_EARN" {
			sawEarn = true
		}
	}
	if !sawEarn {
		t.Fatalf("流水缺 PAYMENT_EARN: %#v", pts)
	}
	portalMvpJSON(t, ts, http.MethodGet, "/api/user/v1/points/tier", "", tok) // 等级端点可达(无等级=null)

	// ---- 7. 发票(缴费后开票,门户可查) ----
	inv := portalMvpJSON(t, ts, http.MethodPost, "/api/user/v1/invoices",
		fmt.Sprintf(`{"billNo":%q}`, billNo), tok)
	if no, _ := inv["invoiceNo"].(string); no == "" {
		t.Fatalf("开票无 invoiceNo: %#v", inv)
	}
	invs := portalMvpJSON(t, ts, http.MethodGet, "/api/user/v1/invoices", "", tok)
	if len(portalMvpArr(t, invs, "records")) == 0 {
		t.Fatalf("发票列表为空: %#v", invs)
	}

	// ---- 8. 工单(报障受理) ----
	fault := portalMvpJSON(t, ts, http.MethodPost, "/api/user/v1/faults",
		`{"faultType":"other","address":"E2E测试市","description":"MVP验收报障"}`, tok)
	ticketNo, _ := fault["ticketNo"].(string)
	if ticketNo == "" {
		t.Fatalf("报障无 ticketNo: %#v", fault)
	}
	faults := portalMvpJSON(t, ts, http.MethodGet, "/api/user/v1/faults", "", tok)
	if !portalMvpArrHas(faults, ticketNo) {
		t.Fatalf("工单列表未见 %s: %#v", ticketNo, faults)
	}

	// ---- 9. 券仓(已用券可见) ----
	cps := portalMvpJSON(t, ts, http.MethodGet, "/api/user/v1/coupons?status=used", "", tok)
	sawUsed := false
	for _, it := range portalMvpArr(t, cps, "items") {
		m, _ := it.(map[string]any)
		if id, _ := m["couponId"].(string); id == couponID {
			sawUsed = true
		}
	}
	if !sawUsed {
		t.Fatalf("券仓未见已核销券 %s: %#v", couponID, cps)
	}
}

// portalMvpArrHas 任意数组字段中存在包含 ticketNo 的项(列表键名差异无关)。
func portalMvpArrHas(data map[string]any, want string) bool {
	for _, v := range data {
		arr, ok := v.([]any)
		if !ok {
			continue
		}
		for _, it := range arr {
			m, _ := it.(map[string]any)
			for _, f := range m {
				if s, _ := f.(string); s == want {
					return true
				}
			}
		}
	}
	return false
}
