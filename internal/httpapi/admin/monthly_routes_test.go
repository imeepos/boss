package adminapi

// 月度填报路由注册断言(T19):14 条路由(12 表级 + regions + summary)。

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/monthly"
)

type fakeMonthly struct{}

func (fakeMonthly) ListRegions(context.Context) ([]monthly.Region, error) { return nil, nil }
func (fakeMonthly) ListUserRevenue(context.Context, monthly.Filter) (monthly.PageResult[monthly.UserRevenue], error) {
	return monthly.PageResult[monthly.UserRevenue]{}, nil
}
func (fakeMonthly) UpsertUserRevenue(context.Context, monthly.UserRevenue) error { return nil }
func (fakeMonthly) ListNetworkDelivery(context.Context, monthly.Filter) (monthly.PageResult[monthly.NetworkDelivery], error) {
	return monthly.PageResult[monthly.NetworkDelivery]{}, nil
}
func (fakeMonthly) UpsertNetworkDelivery(context.Context, monthly.NetworkDelivery) error { return nil }
func (fakeMonthly) ListFinanceCost(context.Context, monthly.Filter) (monthly.PageResult[monthly.FinanceCost], error) {
	return monthly.PageResult[monthly.FinanceCost]{}, nil
}
func (fakeMonthly) UpsertFinanceCost(context.Context, monthly.FinanceCost) error { return nil }
func (fakeMonthly) ImportCSV(context.Context, string, []byte) (monthly.ImportResult, error) {
	return monthly.ImportResult{}, nil
}
func (fakeMonthly) ExportCSV(context.Context, string, string) ([]byte, error) { return nil, nil }
func (fakeMonthly) Summary(context.Context, string) (monthly.Summary, error) {
	return monthly.Summary{}, nil
}

func TestMonthlyRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerMonthlyRoutes(r.Group("/api/admin/v1"), &app.Application{User: &fakeUser{permOk: true}, Monthly: fakeMonthly{}})
	if len(r.Routes()) != 14 {
		t.Fatalf("routes=%d, want 14", len(r.Routes()))
	}
	want := map[string]bool{
		"GET /api/admin/v1/monthly/regions": false, "GET /api/admin/v1/monthly/summary": false,
	}
	for _, tbl := range []string{"user-revenue", "network-delivery", "finance-cost"} {
		want["GET /api/admin/v1/monthly/"+tbl] = false
		want["PUT /api/admin/v1/monthly/"+tbl] = false
		want["POST /api/admin/v1/monthly/"+tbl+"/import"] = false
		want["GET /api/admin/v1/monthly/"+tbl+"/export"] = false
	}
	for _, rt := range r.Routes() {
		k := rt.Method + " " + rt.Path
		if _, ok := want[k]; ok {
			want[k] = true
		}
	}
	for k, hit := range want {
		if !hit {
			t.Fatalf("route missing: %s", k)
		}
	}
}
