package workerapi

// 回归(2026-09-04 任务A-g):扫码异常上报后师傅收到站内消息。
// 现场:scan-abnormal 走 workerAuditOK 纯桩,提交后师傅端无任何可见反馈。

import (
	"context"
	"errors"
	"testing"

	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/worker"
)

// fakeAbnormalLedger 消息台账桩:记录 SendMessage 入参,可注入失败。
type fakeAbnormalLedger struct {
	worker.WorkerLedgerService
	sent    []worker.Message
	sendErr error
}

func (f *fakeAbnormalLedger) SendMessage(_ context.Context, m worker.Message) (int64, error) {
	if f.sendErr != nil {
		return 0, f.sendErr
	}
	f.sent = append(f.sent, m)
	return 1, nil
}

// abnormalRouter 装配扫码异常回归路由(消息台账注入)。
func abnormalRouter(t *testing.T, ledger *fakeAbnormalLedger) map[string]any {
	t.Helper()
	tok := portalGrabToken(t)
	fw := &fakePortalWorkOrder{tickets: []order.DispatchTicket{
		{TicketID: 1, TicketNo: "DT-1", OrderID: 5, WorkerID: 7, Status: "DOING"},
	}}
	r := portalTestRouterWith(t, fw, &fakePortalOrder{}, &fakePortalWorkerSvc{}, ledger)
	return portalWorkerDo(r, "POST", "/api/worker/v1/tickets/DT-1/scan-abnormal", "", tok)
}

func TestScanAbnormalSendsWorkerMessage(t *testing.T) {
	ledger := &fakeAbnormalLedger{}
	res := abnormalRouter(t, ledger)
	if res["code"].(float64) != 0 {
		t.Fatalf("scan-abnormal failed: %v", res)
	}
	if len(ledger.sent) != 1 {
		t.Fatalf("messages=%d, want 1", len(ledger.sent))
	}
	m := ledger.sent[0]
	if m.Level != "WARN" || m.WorkerID != 7 {
		t.Fatalf("message=%+v, want WARN to worker 7", m)
	}
	data, _ := res["data"].(map[string]any)
	if data["messageSent"] != true {
		t.Fatalf("messageSent=%v, want true", data["messageSent"])
	}
}

// 消息写失败不阻断上报主流程(response messageSent=false),失败留 ALERT 日志。
func TestScanAbnormalMessageFailureNotBlocking(t *testing.T) {
	ledger := &fakeAbnormalLedger{sendErr: errors.New("db down")}
	res := abnormalRouter(t, ledger)
	if res["code"].(float64) != 0 {
		t.Fatalf("report must still succeed: %v", res)
	}
	data, _ := res["data"].(map[string]any)
	if data["messageSent"] != false {
		t.Fatalf("messageSent=%v, want false", data["messageSent"])
	}
}
