package adminapi

// 投资测算 handler 测试:信封形状(items)与「未登记=null」语义透传。

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// fakeODNInv 桩 odn.ODNService:仅实现 GridInvestment。
type fakeODNInv struct {
	odn.ODNService
	rows []odn.GridInvestmentRow
	err  error
}

func (f *fakeODNInv) GridInvestment(_ context.Context) ([]odn.GridInvestmentRow, error) {
	return f.rows, f.err
}

func invRouter(f *fakeODNInv) *gin.Engine {
	r := gin.New()
	a := &app.Application{User: &fakeUser{permOk: true}, ODN: f}
	mgr := auth.NewManager("test-secret", time.Hour)
	g := r.Group("/api/admin/v1", middleware.Authn(mgr, auth.AudAdmin))
	registerODNRoutes(g, a)
	return r
}

func TestODNGridInvestmentHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("信封为items且未登记成本为null", func(t *testing.T) {
		f := &fakeODNInv{rows: []odn.GridInvestmentRow{{PrvCode: "PHL001", CityPrefix: "MNL", GridCode: 1, CoverageServed: 3}}}
		w := doJSON(invRouter(f), http.MethodGet, "/api/admin/v1/odn/grid-investment", "")
		if w.Code != http.StatusOK {
			t.Fatalf("HTTP=%d body=%s", w.Code, w.Body.String())
		}
		var body struct {
			Code int `json:"code"`
			Data struct {
				Items []struct {
					PrvCode       string   `json:"prvCode"`
					GridCode      int16    `json:"gridCode"`
					SettledCost   *float64 `json:"settledCost"`
					CostPerServed *float64 `json:"costPerServed"`
				} `json:"items"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v body=%s", err, w.Body.String())
		}
		if body.Code != int(apitypes.CodeOK) || len(body.Data.Items) != 1 {
			t.Fatalf("code=%d items=%d body=%s", body.Code, len(body.Data.Items), w.Body.String())
		}
		row := body.Data.Items[0]
		if row.SettledCost != nil || row.CostPerServed != nil {
			t.Fatalf("未登记成本必须为 null: %v %v", row.SettledCost, row.CostPerServed)
		}
	})

	t.Run("查询失败走错误信封", func(t *testing.T) {
		f := &fakeODNInv{err: context.DeadlineExceeded}
		w := doJSON(invRouter(f), http.MethodGet, "/api/admin/v1/odn/grid-investment", "")
		var body struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if body.Code == int(apitypes.CodeOK) {
			t.Fatalf("期望错误信封,body=%s", w.Body.String())
		}
	})
}
