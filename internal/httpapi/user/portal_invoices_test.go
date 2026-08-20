package userapi

// 门户申请开票单测:真正调用 TaxService 落 invoices;非归属账单 CodeNotFound;重复申请幂等回已有票。

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// newInvoiceRouter 装配带可编程 Tax 桩的门户路由。
func newInvoiceRouter(cust *customer.Customer, tx *fakeTaxStub) (*gin.Engine, *auth.Manager) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("test-secret", time.Hour)
	r := gin.New()
	Register(r, &app.Application{
		Customer:  &userPortalCustSvc{c: cust},
		Billing:   &fakeBilling{},
		Order:     &fakeOrder{},
		Product:   &fakeProduct{},
		WorkOrder: &userPortalWo{},
		Portal:    portal.NewMemory(),
		Channel:   &fakeChannelStub{},
		Tax:       tx,
	}, mgr)
	return r, mgr
}

func applyInvoiceDo(t *testing.T, r *gin.Engine, tok, body string) (int, map[string]any) {
	t.Helper()
	w := userPortalDo(r, http.MethodPost, "/api/user/v1/invoices", body, tok)
	return userPortalCode(t, w)
}

// TestPortal_ApplyInvoice_Issues 开票申请真正落 Tax 域,回传 invoiceNo/invoiceId。
func TestPortal_ApplyInvoice_Issues(t *testing.T) {
	cust := userPortalCust()
	tx := &fakeTaxStub{}
	r, mgr := newInvoiceRouter(cust, tx)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)

	code, data := applyInvoiceDo(t, r, tok, `{"billNo":"B-001"}`)
	if code != int(apitypes.CodeOK) {
		t.Fatalf("issue resp code=%d", code)
	}
	if data["invoiceNo"] != "INV-00000001" || data["invoiceId"] == nil || data["ok"] != true {
		t.Fatalf("issue data=%v", data)
	}
	if len(tx.issuedNo) != 1 || tx.issuedNo[0] != "B-001" {
		t.Fatalf("issuedNo=%v, want [B-001]", tx.issuedNo)
	}
}

// TestPortal_ApplyInvoice_NotFound 账单不存在/非本人归属 → CodeNotFound。
func TestPortal_ApplyInvoice_NotFound(t *testing.T) {
	cust := userPortalCust()
	tx := &fakeTaxStub{issueErr: billing.ErrNotFound}
	r, mgr := newInvoiceRouter(cust, tx)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)

	code, _ := applyInvoiceDo(t, r, tok, `{"billNo":"NOPE"}`)
	if code != int(apitypes.CodeNotFound) {
		t.Fatalf("code=%d, want CodeNotFound", code)
	}
}

// TestPortal_ApplyInvoice_Idempotent 重复申请返回同一张已有票(域层幂等)。
func TestPortal_ApplyInvoice_Idempotent(t *testing.T) {
	cust := userPortalCust()
	tx := &fakeTaxStub{}
	r, mgr := newInvoiceRouter(cust, tx)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)

	for i := 0; i < 2; i++ {
		code, data := applyInvoiceDo(t, r, tok, `{"billNo":"B-001"}`)
		if code != int(apitypes.CodeOK) {
			t.Fatalf("apply#%d code=%d", i, code)
		}
		if data["invoiceNo"] != "INV-00000001" {
			t.Fatalf("apply#%d data=%v", i, data)
		}
	}
}
