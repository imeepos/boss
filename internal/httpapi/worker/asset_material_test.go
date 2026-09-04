package workerapi

// 回归(2026-09-04 任务A-e):工具借还/领料/旧件返库事实落库的班组快照由服务端解析。
// 现场:borrow/give-back 稳定 50000——handler 构造 worker.Tool 未填 GroupID,
// group_id=0 撞 worker_tools_group_id_fkey(23505)。

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/worker"
)

// fakeToolEventSvc 事件服务桩:快照可控,记录 AppendTool 入参。
type fakeToolEventSvc struct {
	worker.WorkerEventService
	snap    *worker.FactSnapshot
	snapErr error
	last    *worker.Tool
}

func (f *fakeToolEventSvc) GetTool(context.Context, int64) (*worker.ToolItem, error) {
	return &worker.ToolItem{ID: 3, Code: "TL-OTDR", Name: "光功率计"}, nil
}

func (f *fakeToolEventSvc) ResolveFactSnapshot(context.Context, int64) (*worker.FactSnapshot, error) {
	if f.snapErr != nil {
		return nil, f.snapErr
	}
	return f.snap, nil
}

func (f *fakeToolEventSvc) AppendTool(_ context.Context, t worker.Tool) (int64, error) {
	cp := t
	f.last = &cp
	return 1, nil
}

// toolBorrowRouter 装配工具借还回归路由。
func toolBorrowRouter(t *testing.T, fe *fakeToolEventSvc) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv("BOSS_JWT_SECRET", "portal-test-secret")
	r := gin.New()
	a := &app.Application{
		WorkerEvent: fe, WorkOrder: &fakePortalWorkOrder{}, Order: &fakePortalOrder{},
		Portal: portal.NewMemory(), QuadLink: &fakePortalQuad{},
	}
	Register(r, a, newWorkerJWTManager())
	return r
}

func TestToolBorrowUsesResolvedSnapshot(t *testing.T) {
	tok := portalGrabToken(t)
	fe := &fakeToolEventSvc{snap: &worker.FactSnapshot{
		WorkerID: 7, GroupID: 6, GroupName: "一队",
		LegalEntityID: 2, LegalEntityName: "主品牌", RegionID: 11, RegionName: "马尼拉",
	}}
	r := toolBorrowRouter(t, fe)
	res := portalWorkerDo(r, "POST", "/api/worker/v1/materials/tools/3/borrow", "", tok)
	if res["code"].(float64) != 0 {
		t.Fatalf("borrow failed: %v", res)
	}
	if fe.last == nil || fe.last.GroupID != 6 || fe.last.GroupName != "一队" ||
		fe.last.LegalEntityID != 2 || fe.last.RegionID != 11 || !fe.last.Borrowed {
		t.Fatalf("AppendTool got %+v, want snapshot-filled borrowed tool", fe.last)
	}
	// give-back:Borrowed=false。
	res = portalWorkerDo(r, "POST", "/api/worker/v1/materials/tools/3/give-back", "", tok)
	if res["code"].(float64) != 0 || fe.last.Borrowed {
		t.Fatalf("give-back failed: %v last=%+v", res, fe.last)
	}
}

// 班组快照不可用 → 40400 明确业务码(不再裸 FK 500)。
func TestToolBorrowGroupInvalid(t *testing.T) {
	tok := portalGrabToken(t)
	fe := &fakeToolEventSvc{snapErr: worker.ErrGroupInvalid}
	r := toolBorrowRouter(t, fe)
	res := portalWorkerDo(r, "POST", "/api/worker/v1/materials/tools/3/borrow", "", tok)
	if res["code"].(float64) != 40400 {
		t.Fatalf("code=%v, want 40400", res["code"])
	}
	if fe.last != nil {
		t.Fatal("AppendTool must not be invoked when snapshot unavailable")
	}
}
