package userapi

// 契约:GET /payments(我的缴费记录)
// 默认仅 SUCCESS;FAILED/REFUNDED 经 ?include=failed,refunded 才回传。
// 业务口径:terms.md §4 payment.status 枚举;UI 默认隐藏失败/退款避免误读。
// 状态字段无论是否过滤都带回,便于前端识别"被过滤掉"的项。

import (
	"net/http"
	"testing"

	"github.com/ymm-001/boss/internal/domain/billing"
)

func TestPortal_PaymentListFilter(t *testing.T) {
	cust, _ := actionCtx()
	pays := []billing.Payment{
		{PayNo: "PAY-OK", BillID: 1, CustomerID: cust.ID, Amount: 99, Method: "wechat", Status: billing.PaymentStatusSuccess},
		{PayNo: "PAY-FAIL", BillID: 0, CustomerID: cust.ID, Amount: 50, Method: "card", Status: billing.PaymentStatusFailed},
		{PayNo: "PAY-RFND", BillID: 2, CustomerID: cust.ID, Amount: 80, Method: "alipay", Status: billing.PaymentStatusRefunded},
	}
	bill := &billingWithPays{pays: pays}
	r, mgr := newActionsRouter(&woWithDispatch{}, bill, nil, nil, nil, nil, nil, nil)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)

	t.Run("默认仅 SUCCESS", func(t *testing.T) {
		w := userPortalDo(r, http.MethodGet, "/api/user/v1/payments", ``, tok)
		code, data := userPortalCode(t, w)
		if code != 0 {
			t.Fatalf("status=%d body=%s", code, w.Body.String())
		}
		items, _ := data["items"].([]any)
		if len(items) != 1 {
			t.Fatalf("期望 1 条 SUCCESS,实得 %d: %+v", len(items), items)
		}
		first, _ := items[0].(map[string]any)
		if first["payNo"] != "PAY-OK" || first["status"] != billing.PaymentStatusSuccess {
			t.Fatalf("PAY-OK 应出现: %+v", first)
		}
	})

	t.Run("include=failed 带回 FAILED", func(t *testing.T) {
		w := userPortalDo(r, http.MethodGet, "/api/user/v1/payments?include=failed", ``, tok)
		code, data := userPortalCode(t, w)
		if code != 0 {
			t.Fatalf("status=%d body=%s", code, w.Body.String())
		}
		items, _ := data["items"].([]any)
		if len(items) != 2 {
			t.Fatalf("期望 2 条 SUCCESS+FAILED,实得 %d", len(items))
		}
	})

	t.Run("include=failed,refunded 全回传", func(t *testing.T) {
		w := userPortalDo(r, http.MethodGet, "/api/user/v1/payments?include=failed,refunded", ``, tok)
		code, data := userPortalCode(t, w)
		if code != 0 {
			t.Fatalf("status=%d body=%s", code, w.Body.String())
		}
		items, _ := data["items"].([]any)
		if len(items) != 3 {
			t.Fatalf("期望 3 条全回传,实得 %d", len(items))
		}
	})

	t.Run("include=refunded 仅带回 REFUNDED", func(t *testing.T) {
		w := userPortalDo(r, http.MethodGet, "/api/user/v1/payments?include=refunded", ``, tok)
		code, data := userPortalCode(t, w)
		if code != 0 {
			t.Fatalf("status=%d body=%s", code, w.Body.String())
		}
		items, _ := data["items"].([]any)
		if len(items) != 2 {
			t.Fatalf("期望 2 条 SUCCESS+REFUNDED,实得 %d", len(items))
		}
	})

	t.Run("未知 include 项不影响默认行为", func(t *testing.T) {
		w := userPortalDo(r, http.MethodGet, "/api/user/v1/payments?include=garbage", ``, tok)
		code, data := userPortalCode(t, w)
		if code != 0 {
			t.Fatalf("status=%d body=%s", code, w.Body.String())
		}
		items, _ := data["items"].([]any)
		if len(items) != 1 {
			t.Fatalf("期望 1 条 SUCCESS,实得 %d", len(items))
		}
	})
}
