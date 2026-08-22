// 回归:入驻申请提交成功 → 消息中心落 todo 待办(Notify.Emit);幂等键 partner_apply/{id}。
package adminapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/partner"
)

// stubPartnerService 只实现 Submit,其余方法不可达。
type stubPartnerService struct{ id int64 }

func (s *stubPartnerService) Submit(_ context.Context, _ partner.Application) (int64, error) {
	return s.id, nil
}
func (s *stubPartnerService) ListApplications(context.Context, string) ([]partner.Application, error) {
	return nil, nil
}
func (s *stubPartnerService) Approve(context.Context, int64, int64) (partner.ApproveResult, error) {
	return partner.ApproveResult{}, nil
}
func (s *stubPartnerService) Reject(context.Context, int64, int64, string) error { return nil }
func (s *stubPartnerService) Profile(context.Context, int64) (partner.PartnerProfile, error) {
	return partner.PartnerProfile{}, nil
}
func (s *stubPartnerService) ListStaff(context.Context, int64) ([]partner.StaffRow, error) {
	return nil, nil
}
func (s *stubPartnerService) CreateStaff(context.Context, int64, string, string, string, string) (int64, error) {
	return 0, nil
}
func (s *stubPartnerService) SetStaffStatus(context.Context, int64, int64, int16) error { return nil }
func (s *stubPartnerService) ListOrders(context.Context, int64) ([]partner.OrderRow, error) {
	return nil, nil
}

func TestPartnerSubmitEmitsTodo(t *testing.T) {
	ns := notify.NewMemStore()
	a := &app.Application{Partner: &stubPartnerService{id: 77}, Notify: ns}
	r := gin.New()
	api := r.Group("/api/admin/v1")
	registerPartnerPublicRoutes(api, a)

	body := `{"companyName":"ACME","creditCode":"91310000MA1FL8X00A","contactName":"张三",` +
		`"contactPhone":"13800138000","email":"a@b.co","businessDesc":"宽带合作"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/partner/applications", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	items, total, err := ns.List(context.Background(), "sysadmin", 1, notify.Filter{})
	if err != nil || total != 1 {
		t.Fatalf("list total=%d err=%v", total, err)
	}
	it := items[0]
	if it.Category != notify.CategoryTodo || it.RefType != "partner_apply" || it.RefID != "77" {
		t.Fatalf("item=%+v", it)
	}
	if it.Title == "" || it.Link != "/org/partner" {
		t.Fatalf("title/link empty: %+v", it)
	}
}
