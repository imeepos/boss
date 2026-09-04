package workerapi

// 回归(2026-09-04 任务A-d):激活失败尝试落痕,lastTry 可见。
// 现场:师傅端 GET activation 的 lastTry 恒为空串——激活 FAILED 回执已落库但查询从不读。

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/portal"
)

// fakeActivationLedger 激活台账桩:记录 FAILED 尝试,返回固定最近尝试。
type fakeActivationLedger struct {
	order.OrderLedgerService
	failed []int64
	latest *order.ActivationCallback
}

func (f *fakeActivationLedger) AppendActivationCallback(_ context.Context, c order.ActivationCallback) (int64, error) {
	if c.Result == "FAILED" {
		f.failed = append(f.failed, c.OrderID)
	}
	return 1, nil
}

func (f *fakeActivationLedger) LatestActivationCallback(context.Context, int64) (*order.ActivationCallback, error) {
	return f.latest, nil
}

// activationLedgerRouter 装配带激活台账桩的路由。
func activationLedgerRouter(t *testing.T, fo *fakePortalOrder, ledger *fakeActivationLedger) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv("BOSS_JWT_SECRET", "portal-test-secret")
	fw := &fakePortalWorkOrder{tickets: []order.DispatchTicket{
		{TicketID: 1, TicketNo: "DT-1", OrderID: 5, WorkerID: 7, Status: "DOING"},
	}}
	r := gin.New()
	a := &app.Application{
		WorkOrder: fw, Order: fo, OrderLedger: ledger,
		Portal: portal.NewMemory(), QuadLink: &fakePortalQuad{},
		Automation: app.NewAutomation(fo, nil),
	}
	Register(r, a, newWorkerJWTManager())
	return r
}

// GET activation:lastTry 返回最近一次激活尝试时刻(RFC3339)。
func TestActivationLastTryVisible(t *testing.T) {
	tok := portalGrabToken(t)
	tried := time.Now()
	ledger := &fakeActivationLedger{latest: &order.ActivationCallback{
		ID: 9, OrderID: 5, Result: "FAILED", TriedAt: &tried,
	}}
	fo := &fakePortalOrder{stage: 9}
	r := activationLedgerRouter(t, fo, ledger)
	res := portalWorkerDo(r, "GET", "/api/worker/v1/tickets/DT-1/activation", "", tok)
	data, _ := res["data"].(map[string]any)
	if data["lastTry"].(string) != tried.Format(time.RFC3339) {
		t.Fatalf("lastTry=%v, want %s", data["lastTry"], tried.Format(time.RFC3339))
	}
	if data["status"].(string) != "PENDING" {
		t.Fatalf("status=%v, want PENDING at stage 9", data["status"])
	}
}

// 激活腿失败(activateUser/notifyActivation)→ FAILED 尝试落痕。
func TestActivationFailureRecorded(t *testing.T) {
	tok := portalGrabToken(t)
	ledger := &fakeActivationLedger{}
	fo := &fakePortalOrder{stage: 9, activateErr: errors.New("boom")}
	r := activationLedgerRouter(t, fo, ledger)
	portalWorkerDo(r, "POST", "/api/worker/v1/tickets/DT-1/activate", "", tok)
	if len(ledger.failed) != 1 || ledger.failed[0] != 5 {
		t.Fatalf("failed attempts=%v, want [5]", ledger.failed)
	}
}

// 非激活腿失败(updateMap)→ 不计 FAILED(激活本身已成功)。
func TestUpdateMapFailureNotRecordedAsActivation(t *testing.T) {
	tok := portalGrabToken(t)
	ledger := &fakeActivationLedger{}
	fo := &fakePortalOrder{stage: 9, noEffect: false}
	r := activationLedgerRouter(t, fo, ledger)
	// ActivateUser/Notify/UpdateMap 全成功桩 → 无失败可记。
	portalWorkerDo(r, "POST", "/api/worker/v1/tickets/DT-1/activate", "", tok)
	if len(ledger.failed) != 0 {
		t.Fatalf("failed attempts=%v, want empty on success path", ledger.failed)
	}
}
