package tl1_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ymm-001/boss/cmd/tl1sim/sim"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/domain/provision/tl1"
)

// ④ 环节10 三态:UP 在线放行;Power-Off 不伪造成功;查无 ONU 报未开通。
func TestExecActivateUserStates(t *testing.T) {
	ctx := context.Background()
	// UP + CFGSTAT=NORMAL → nil
	_, m, _ := startSim(t, nil)
	ex := newExec(m, testParams())
	if err := ex.Exec(ctx, taskOf("T-1010", tl1.StagePreConfigOLT)); err != nil {
		t.Fatalf("preConfig: %v", err)
	}
	if err := ex.Exec(ctx, taskOf("T-1010", tl1.StageActivateUser)); err != nil {
		t.Fatalf("activate UP: %v", err)
	}

	// Power-Off → error "onu not online: Power-Off"
	_, m2, _ := startSim(t, func(o *sim.Options) { o.ONUOperState = "Power-Off" })
	ex2 := newExec(m2, testParams())
	if err := ex2.Exec(ctx, taskOf("T-1011", tl1.StagePreConfigOLT)); err != nil {
		t.Fatalf("preConfig: %v", err)
	}
	err := ex2.Exec(ctx, taskOf("T-1011", tl1.StageActivateUser))
	if err == nil || !strings.Contains(err.Error(), "onu not online: Power-Off") {
		t.Fatalf("want not-online err, got %v", err)
	}

	// 查无 ONU → error "onu not provisioned"
	_, m3, _ := startSim(t, nil)
	ex3 := newExec(m3, testParams())
	err = ex3.Exec(ctx, taskOf("T-1012", tl1.StageActivateUser))
	if err == nil || !strings.Contains(err.Error(), "onu not provisioned") {
		t.Fatalf("want not-provisioned err, got %v", err)
	}
}

// ⑤ 环节11:全部业务流 DESC 命中 → nil;缺任一条 → error。
func TestExecNotifyActivation(t *testing.T) {
	ctx := context.Background()
	_, m, _ := startSim(t, nil)
	ex := newExec(m, testParams())
	if err := ex.Exec(ctx, taskOf("T-1020", tl1.StagePreConfigOLT)); err != nil {
		t.Fatalf("preConfig: %v", err)
	}
	if err := ex.Exec(ctx, taskOf("T-1020", tl1.StageNotifyActivation)); err != nil {
		t.Fatalf("notify all-hit: %v", err)
	}

	// 缺业务流:仅建 ONU 不建流。
	_, m2, _ := startSim(t, nil)
	p := testParams()
	cl := tl1.NewClient(m2)
	if err := cl.AddONU(ctx, testAddONUParams(p)); err != nil {
		t.Fatalf("AddONU: %v", err)
	}
	ex2 := newExec(m2, p)
	err := ex2.Exec(ctx, taskOf("T-1021", tl1.StageNotifyActivation))
	if err == nil || !strings.Contains(err.Error(), "service port missing") {
		t.Fatalf("want missing err, got %v", err)
	}
}

// testAddONUParams 组装 AddONUParams(测试专用,镜像 executor 内部映射)。
func testAddONUParams(p tl1.Params) tl1.AddONUParams {
	return tl1.AddONUParams{
		OLTID: p.OLTID, PONID: p.PONID, AuthType: p.AuthType, ONUID: p.ONUID,
		ONUNo: p.ONUNo, ONUType: p.ONUType, Desc: p.Desc,
	}
}

// ⑦ Manager 断线重连一次后整段重跑成功,零重复写。
func TestExecReconnectRerun(t *testing.T) {
	srv, m, rec := startSim(t, nil)
	ex := newExec(m, testParams())
	task := taskOf("T-1030", tl1.StagePreConfigOLT)
	ctx := context.Background()
	if err := ex.Exec(ctx, task); err != nil {
		t.Fatalf("first Exec: %v", err)
	}
	srv.DropConns() // 网元侧断线,客户端持死连接
	if err := ex.Exec(ctx, task); err != nil {
		t.Fatalf("rerun after drop: %v", err)
	}
	assertCounts(t, recordVerbs(t, rec), 1, 2)
}

// 兜底:不支持的环节事件明确报 unsupported,不静默。
func TestExecUnsupportedStage(t *testing.T) {
	_, m, _ := startSim(t, nil)
	ex := newExec(m, testParams())
	err := ex.Exec(context.Background(), taskOf("T-1099", "unknownStage"))
	if err == nil || !strings.Contains(err.Error(), "unsupported stage event") {
		t.Fatalf("want unsupported err, got %v", err)
	}
}

// 编译期守卫:Executor 满足 provision.Executor 口(T5 装配依赖)。
var _ provision.Executor = (*tl1.Executor)(nil)
