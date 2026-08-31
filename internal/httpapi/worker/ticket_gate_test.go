package workerapi

// 接单半径闸门回归:radiusKm>0 且工单坐标快照 + 师傅最新位置齐备时超距拒;
// 前提缺失(无快照/无位置/未设半径)一律跳过,不得误拦。

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/worker"
)

// fakeGateLocation 桩位置服务:按 workerID 返回可配置最新位置。
type fakeGateLocation struct {
	loc map[int64]*worker.Location
}

func (f *fakeGateLocation) ReportLocation(context.Context, worker.Location) error { return nil }
func (f *fakeGateLocation) LatestLocation(_ context.Context, id int64) (*worker.Location, error) {
	return f.loc[id], nil
}
func (f *fakeGateLocation) LatestLocationForOrder(context.Context, int64) (*worker.Location, error) {
	return nil, nil
}
func (f *fakeGateLocation) ListLocationHistory(context.Context, int64, time.Time, int) ([]worker.Location, error) {
	return nil, nil
}

// TestHaversineM 契约:马尼拉→奎松约 10km 量级;同点距离为 0。
func TestHaversineM(t *testing.T) {
	if d := haversineM(14.599, 120.984, 14.599, 120.984); d > 0.01 {
		t.Fatalf("same point: %v", d)
	}
	d := haversineM(14.599, 120.984, 14.6800, 121.0360) // 马尼拉→奎松 ~11km
	if d < 9000 || d > 13000 {
		t.Fatalf("manila→quezon: %v", d)
	}
}

// TestRadiusGate 契约:5km 半径内放行,50km 超距拒;无快照/无位置跳过。
func TestRadiusGate(t *testing.T) {
	manila := &order.DispatchTicket{TicketNo: "TK-1", WorkerID: 0, Status: "PENDING",
		RegionID: 1, SiteLat: ptrF(14.599), SiteLng: ptrF(120.984)}
	settings := map[int64]*worker.Settings{
		7: {WorkerID: 7, Accepting: true, RadiusKm: 5, AcceptTypes: ""},
	}
	mk := func(t *testing.T, loc *worker.Location, tk *order.DispatchTicket) map[string]any {
		t.Helper()
		tok := portalGrabToken(t)
		fw := &fakePortalWorkOrder{tickets: []order.DispatchTicket{*tk}}
		gate := &fakeGateLocation{}
		if loc != nil {
			gate.loc = map[int64]*worker.Location{7: loc}
		}
		gin.SetMode(gin.TestMode)
		t.Setenv("BOSS_JWT_SECRET", "portal-test-secret")
		r := gin.New()
		a := &app.Application{
			Worker:         &fakePortalWorkerSvc{region: 1},
			WorkOrder:      fw,
			Order:          &fakePortalOrder{},
			WorkerLedger:   &fakePortalLedger{settings: settings},
			WorkerLocation: gate,
			Portal:         portal.NewMemory(),
			QuadLink:       &fakePortalQuad{},
			Automation:     app.NewAutomation(&fakePortalOrder{}, nil),
		}
		Register(r, a, newWorkerJWTManager())
		return portalWorkerDo(r, "POST", "/api/worker/v1/hall/"+tk.TicketNo+"/grab", "", tok)
	}

	t.Run("半径内放行", func(t *testing.T) {
		res := mk(t, &worker.Location{Lat: 14.605, Lng: 120.990}, manila) // ~1km
		if res["code"].(float64) != 0 {
			t.Fatalf("in-radius grab should pass: %v", res)
		}
	})
	t.Run("超距拒绝", func(t *testing.T) {
		res := mk(t, &worker.Location{Lat: 14.1700, Lng: 121.4000}, manila) // ~70km
		if res["code"].(float64) == 0 {
			t.Fatalf("out-of-radius grab should be rejected: %v", res)
		}
	})
	t.Run("无位置跳过", func(t *testing.T) {
		res := mk(t, nil, manila)
		if res["code"].(float64) != 0 {
			t.Fatalf("no location must not block: %v", res)
		}
	})
	t.Run("无快照跳过", func(t *testing.T) {
		noCoord := &order.DispatchTicket{TicketNo: "TK-2", WorkerID: 0, Status: "PENDING", RegionID: 1}
		res := mk(t, &worker.Location{Lat: 0, Lng: 0}, noCoord)
		if res["code"].(float64) != 0 {
			t.Fatalf("no snapshot must not block: %v", res)
		}
	})
}

// ptrF float64 取指(测试小工具)。
func ptrF(v float64) *float64 { return &v }
