package workerapi

// 回归(2026-09-04 任务A-f):签到 GPS 闸门——无坐标工单显式降级不静默,
// 有坐标缺定位拒绝,超 100m 拒绝。

import (
	"fmt"
	"testing"

	"github.com/ymm-001/boss/internal/domain/order"
)

// jsonPos 组签到定位体;dq=双引号字符(免转义构造)。
func jsonPos(lat, lng float64) string {
	dq := string(rune(34))
	return "{" + dq + "lat" + dq + ":" + fmt.Sprintf("%.6f", lat) +
		"," + dq + "lng" + dq + ":" + fmt.Sprintf("%.6f", lng) + "}"
}

// checkinDo 以给定站点快照与签到定位发起签到请求。
func checkinDo(t *testing.T, siteLat, siteLng, lat, lng *float64) map[string]any {
	t.Helper()
	tok := portalGrabToken(t)
	fw := &fakePortalWorkOrder{tickets: []order.DispatchTicket{
		{TicketID: 1, TicketNo: "DT-1", OrderID: 5, WorkerID: 7, Status: "DOING",
			SiteLat: siteLat, SiteLng: siteLng},
	}}
	r := portalTestRouter(t, fw, &fakePortalOrder{})
	body := ""
	if lat != nil && lng != nil {
		body = jsonPos(*lat, *lng)
	}
	return portalWorkerDo(r, "POST", "/api/worker/v1/tickets/DT-1/checkin", body, tok)
}

func TestCheckinGPSGate(t *testing.T) {
	t.Run("无坐标工单 → 显式降级口径写入响应", func(t *testing.T) {
		res := checkinDo(t, nil, nil, nil, nil)
		if res["code"].(float64) != 0 {
			t.Fatalf("degraded checkin should pass: %v", res)
		}
		data, _ := res["data"].(map[string]any)
		if data["gpsCheck"] != "SKIPPED_NO_SITE_COORDS" {
			t.Fatalf("gpsCheck=%v, want SKIPPED_NO_SITE_COORDS", data["gpsCheck"])
		}
	})

	t.Run("有坐标且就近 → OK", func(t *testing.T) {
		lat, lng := 14.5995, 120.9842 // 距站点约 60m
		siteLat, siteLng := 14.599, 120.984
		res := checkinDo(t, &siteLat, &siteLng, &lat, &lng)
		if res["code"].(float64) != 0 {
			t.Fatalf("nearby checkin failed: %v", res)
		}
		data, _ := res["data"].(map[string]any)
		if data["gpsCheck"] != "OK" {
			t.Fatalf("gpsCheck=%v, want OK", data["gpsCheck"])
		}
	})

	t.Run("有坐标但超 100m → 40900", func(t *testing.T) {
		lat, lng := 14.609, 120.984 // 距站点约 1.1km
		siteLat, siteLng := 14.599, 120.984
		res := checkinDo(t, &siteLat, &siteLng, &lat, &lng)
		if res["code"].(float64) != 40900 {
			t.Fatalf("code=%v, want 40900", res["code"])
		}
	})

	t.Run("有坐标但请求缺定位 → 42200", func(t *testing.T) {
		siteLat, siteLng := 14.599, 120.984
		res := checkinDo(t, &siteLat, &siteLng, nil, nil)
		if res["code"].(float64) != 42200 {
			t.Fatalf("code=%v, want 42200", res["code"])
		}
	})
}
