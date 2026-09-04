package workerapi

// 回归(2026-09-04 任务A):签收闸门与真实状态机推进。
// 现场:POST /tickets/{no}/sign 曾是纯审计桩——扫码未一致/激活未成功仍返回
// {ok:true} 而状态机不流转(假成功)。修复后:闸门(9/10/11)不满足回 40910+reason;
// 满足才 updateMap 真推进,0 行生效显性 5xx+[fake-success] ALERT。

import (
	"testing"

	"github.com/ymm-001/boss/internal/domain/order"
)

// signRouter 装配签收回归路由;返回响应与订单桩。
func signRouter(t *testing.T, stage int8, noEffect bool) (map[string]any, *fakePortalOrder) {
	t.Helper()
	tok := portalGrabToken(t)
	fw := &fakePortalWorkOrder{tickets: []order.DispatchTicket{
		{TicketID: 1, TicketNo: "DT-1", OrderID: 5, WorkerID: 7, Status: "DOING"},
	}}
	fo := &fakePortalOrder{stage: stage, noEffect: noEffect}
	r := portalTestRouter(t, fw, fo)
	res := portalWorkerDo(r, "POST", "/api/worker/v1/tickets/DT-1/sign", `{}`, tok)
	return res, fo
}

func TestWorkerSignGate(t *testing.T) {
	cases := []struct {
		stage  int8
		reason string
	}{
		{8, "扫码绑定未完成,不可签收"},
		{9, "激活上报未提交,不可签收"},
		{10, "激活未成功,不可签收"},
	}
	for _, tc := range cases {
		res, fo := signRouter(t, tc.stage, false)
		if res["code"].(float64) != 40910 {
			t.Fatalf("stage %d: code=%v, want 40910", tc.stage, res["code"])
		}
		if fo.updated != 0 {
			t.Fatalf("stage %d: 状态机不得推进", tc.stage)
		}
		data, _ := res["data"].(map[string]any)
		if data == nil || data["reason"] != tc.reason {
			t.Fatalf("stage %d: reason=%v, want %q", tc.stage, data, tc.reason)
		}
	}
}

// 闸门全过(stage 11)→ updateMap 真推进到 12,返回 ok+stage 12。
func TestWorkerSignAdvancesStateMachine(t *testing.T) {
	res, fo := signRouter(t, 11, false)
	if res["code"].(float64) != 0 {
		t.Fatalf("sign failed: %v", res)
	}
	if fo.updated != 5 {
		t.Fatalf("updateMap not invoked, updated=%d", fo.updated)
	}
	data, _ := res["data"].(map[string]any)
	if data["stage"].(float64) != 12 {
		t.Fatalf("stage=%v, want 12", data["stage"])
	}
}

// 0 行生效守卫:updateMap 后 stage 仍 11 → 显性 5xx,不得假成功。
func TestWorkerSignNoEffectGuard(t *testing.T) {
	res, fo := signRouter(t, 11, true)
	if res["code"].(float64) == 0 {
		t.Fatalf("no-effect sign must not be ok: %v", res)
	}
	if fo.updated != 0 {
		t.Fatal("no-effect stub should not record update")
	}
}

// 已在环节12(自动化链路已完成)→ 幂等成功,不再重复推进。
func TestWorkerSignIdempotentAtDone(t *testing.T) {
	res, fo := signRouter(t, 12, false)
	if res["code"].(float64) != 0 {
		t.Fatalf("idempotent sign failed: %v", res)
	}
	if fo.updated != 0 {
		t.Fatal("stage 12 sign must not re-invoke updateMap")
	}
}
