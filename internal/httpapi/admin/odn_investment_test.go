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

// fakeODNInv 桩 odn.ODNService:仅实现投资测算与分光容量读路径。
type fakeODNInv struct {
	odn.ODNService
	rows       []odn.GridInvestmentRow
	cityRows   []odn.CityInvestmentRow
	capacity   *odn.SplitCapacityReport
	backfill   *odn.SplitBackfillResult
	backfilled int
	err        error
}

func (f *fakeODNInv) GridInvestment(_ context.Context) ([]odn.GridInvestmentRow, error) {
	return f.rows, f.err
}

func (f *fakeODNInv) CityInvestment(_ context.Context) ([]odn.CityInvestmentRow, error) {
	return f.cityRows, f.err
}

func (f *fakeODNInv) SplitCapacity(_ context.Context) (*odn.SplitCapacityReport, error) {
	return f.capacity, f.err
}

func (f *fakeODNInv) BackfillSplitCapacity(_ context.Context) (*odn.SplitBackfillResult, error) {
	f.backfilled++
	return f.backfill, f.err
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
func TestODNCityInvestmentHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	potential, connected, expandable := 64, 3, 61
	f := &fakeODNInv{cityRows: []odn.CityInvestmentRow{{
		PrvCode: "PHL001", CityPrefix: "MNL", GridCount: 2, CoverageServed: 5,
		PotentialHomes: &potential, ConnectedHomes: &connected, ExpandableHomes: &expandable,
	}}}
	w := doJSON(invRouter(f), http.MethodGet, "/api/admin/v1/odn/city-investment", "")
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				CityPrefix      string   `json:"cityPrefix"`
				PotentialHomes  *int     `json:"potentialHomes"`
				ExpandableHomes *int     `json:"expandableHomes"`
				SettledCost     *float64 `json:"settledCost"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if len(body.Data.Items) != 1 {
		t.Fatalf("items=%d body=%s", len(body.Data.Items), w.Body.String())
	}
	row := body.Data.Items[0]
	if row.PotentialHomes == nil || *row.PotentialHomes != 64 || *row.ExpandableHomes != 61 {
		t.Fatalf("容量列透传错误: %+v", row)
	}
	if row.SettledCost != nil {
		t.Fatalf("未登记成本必须为 null: %v", *row.SettledCost)
	}
}

func TestODNSplitCapacityHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := &fakeODNInv{capacity: &odn.SplitCapacityReport{
		Items:   []odn.SplitCapacityRow{{DeviceID: 7, Code: "OBD001", Kind: "OBD", SplitLevel: 1, Ratio: 8, UsedPorts: 2, Expandable: 6}},
		Summary: odn.SplitCapacitySummary{Devices: 1, PotentialHomes: 8, ConnectedHomes: 2, ExpandableHomes: 6},
	}}
	w := doJSON(invRouter(f), http.MethodGet, "/api/admin/v1/odn/split-capacity", "")
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				Code       string `json:"code"`
				Expandable int    `json:"expandable"`
			} `json:"items"`
			Summary struct {
				PotentialHomes int `json:"potentialHomes"`
			} `json:"summary"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if len(body.Data.Items) != 1 || body.Data.Items[0].Expandable != 6 || body.Data.Summary.PotentialHomes != 8 {
		t.Fatalf("容量报告形状错误: %s", w.Body.String())
	}
}

func TestODNBackfillSplitHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := &fakeODNInv{backfill: &odn.SplitBackfillResult{DevicesModeled: 3, Level1: 2, Level2: 1}}
	w := doJSON(invRouter(f), http.MethodPost, "/api/admin/v1/odn/resource-chains/backfill-split", "")
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Result struct {
				DevicesModeled int `json:"devicesModeled"`
			} `json:"result"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Code != int(apitypes.CodeOK) || body.Data.Result.DevicesModeled != 3 {
		t.Fatalf("回写结果错误: %s", w.Body.String())
	}
	if f.backfilled != 1 {
		t.Fatalf("域方法应恰好调用一次,得 %d", f.backfilled)
	}
}
