package adminapi

// 契约:指派成功后触发 PushNotifier.NotifyWorker(尽力而为,nil 安全)。

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	pushdomain "github.com/ymm-001/boss/internal/domain/push"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

type fakePushNotifier struct {
	mu    sync.Mutex
	calls []fakePushCall
}

type fakePushCall struct {
	workerID int64
	title    string
	alert    string
	extras   map[string]string
}

func (f *fakePushNotifier) NotifyWorker(_ context.Context, workerID int64, title, alert string, extras map[string]string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fakePushCall{workerID, title, alert, extras})
	return nil
}

func (f *fakePushNotifier) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// waitForCond 通知为 goroutine 异步,轮询等待至条件成立。
func waitForCond(t *testing.T, cond func() bool) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}

// newDispatchRouterWithNotifier 注入 PushNotifier 的装配;nil 走接口零值(typed-nil 陷阱)。
func newDispatchRouterWithNotifier(notify pushdomain.WorkerNotifier, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	wo := &fakeDispatchOrder{}
	ws := &fakeWorkerSvc{w: &worker.Worker{ID: 5, Name: "张师傅", Status: 1}}
	r := gin.New()
	Register(r, &app.Application{
		User: &fakeUser{permOk: true}, WorkOrder: wo, OrderLedger: &fakeOrderLedger{},
		Worker: ws, PushNotifier: notify,
	}, mgr)
	return r
}

func TestAssignTicketNotifiesWorker(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	notify := &fakePushNotifier{}
	r := newDispatchRouterWithNotifier(notify, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/dispatch/pool/TK-1/assign", `{"masterId":5}`, authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	waitForCond(t, func() bool { return notify.count() == 1 })
	notify.mu.Lock()
	c := notify.calls[0]
	notify.mu.Unlock()
	if c.workerID != 5 || c.title != "新派工单" || c.extras["ticketNo"] != "TK-1" {
		t.Fatalf("notify call=%+v", c)
	}
	if c.alert != "工单 TK-1 已派给张师傅,请打开师傅端查看详情" {
		t.Fatalf("alert=%q", c.alert)
	}
}

func TestAssignTicketNilNotifierSafe(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	r := newDispatchRouterWithNotifier(nil, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/dispatch/pool/TK-1/assign", `{"masterId":5}`, authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("nil notifier should not break assign: %d %s", w.Code, w.Body.String())
	}
}
