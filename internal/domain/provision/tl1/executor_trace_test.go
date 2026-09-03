package tl1_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ymm-001/boss/cmd/tl1sim/sim"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/domain/provision/tl1"
)

// ---- trace 收口断言:指令与应答 ctag 逐条对齐 ----

// respChunks 按 joinResp 的 4 横线分隔约定切应答段(与设备表格 5 横线不冲突)。
func respChunks(resp string) []string {
	return strings.Split(resp, "\n----\n")
}

// cmdCTag 取指令行 ctag 位:Build 产物 "<VERB>::<access>:<ctag>::<payload>;" 第 4 冒号段。
func cmdCTag(line string) string {
	parts := strings.Split(strings.TrimSuffix(line, ";"), ":")
	if len(parts) < 4 {
		return ""
	}
	return parts[3]
}

// mLine 应答段内首条 M 行:返回 (ctag, completion);无 M 行 ok=false。
func mLine(chunk string) (ctag, compl string, ok bool) {
	for _, ln := range strings.Split(chunk, "\n") {
		f := strings.Fields(ln)
		if len(f) == 3 && f[0] == "M" {
			return f[1], f[2], true
		}
	}
	return "", "", false
}

// assertAligned 成功留痕硬断言:指令与应答段 1:1 且同序 ctag 对齐,
// 每段 COMPLD,业务指令(ADD-)段内 EN=0。
func assertAligned(t *testing.T, trace provision.ExecTrace) {
	t.Helper()
	chunks := respChunks(trace.Response)
	if len(trace.Commands) != len(chunks) {
		t.Fatalf("commands=%d response chunks=%d, want 1:1\ncmds:\n%s\nresp:\n%s",
			len(trace.Commands), len(chunks), strings.Join(trace.Commands, "\n"), trace.Response)
	}
	for i, cmd := range trace.Commands {
		ctag, compl, ok := mLine(chunks[i])
		if !ok {
			t.Fatalf("chunk %d has no M line (cmd=%s):\n%s", i, cmd, chunks[i])
		}
		if ctag != cmdCTag(cmd) {
			t.Fatalf("chunk %d ctag=%s != cmd ctag=%s (cmd=%s)", i, ctag, cmdCTag(cmd), cmd)
		}
		if compl != "COMPLD" {
			t.Fatalf("chunk %d completion=%s, want COMPLD (cmd=%s)", i, compl, cmd)
		}
		if strings.HasPrefix(cmd, "ADD-") && !strings.Contains(chunks[i], "EN=0") {
			t.Fatalf("business cmd %s response missing EN=0:\n%s", cmd, chunks[i])
		}
	}
}

// assertNoErrorResidue 成功留痕不得残留连接错误、错误收尾或 TRC 占位 ctag。
func assertNoErrorResidue(t *testing.T, trace provision.ExecTrace) {
	t.Helper()
	for _, bad := range []string{"connection broken", "EXEC-ERROR", "TRC"} {
		if strings.Contains(trace.Response, bad) {
			t.Fatalf("success response contains %q:\n%s", bad, trace.Response)
		}
		for _, cmd := range trace.Commands {
			if strings.Contains(cmd, bad) {
				t.Fatalf("success command contains %q: %s", bad, cmd)
			}
		}
	}
}

// ⑨ 正常多命令 trace:每条指令 ctag 与应答 M ctag 一一对齐,业务命令全 COMPLD/EN=0。
func TestExecTraceCommandsAlignedOnSuccess(t *testing.T) {
	_, m, _ := startSim(t, nil)
	ex := newExec(m, testParams())
	trace, err := ex.Exec(context.Background(), taskOf("T-1060", tl1.StagePreConfigOLT))
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if len(trace.Commands) == 0 {
		t.Fatal("no commands recorded")
	}
	assertAligned(t, trace)
	assertNoErrorResidue(t, trace)
	joined := strings.Join(trace.Commands, "\n")
	for _, tag := range []string{"ADDONT", "Internet", "TR069"} {
		if !strings.Contains(joined, ":"+tag+":") {
			t.Fatalf("business ctag %s missing in commands:\n%s", tag, joined)
		}
	}
}

// ⑩ 断线重连后最终成功:最终 trace 只留最终尝试(指令与应答 1:1),
// 被重试替代的 broken pipe 不作为成功证据混入。
// 先以 ActivateUser 暖出缓存会话(查空报错但会话保留),再 DropConns 触发重连。
func TestExecTraceReconnectSuccessClean(t *testing.T) {
	srv, m, _ := startSim(t, nil)
	ex := newExec(m, testParams())
	ctx := context.Background()
	if _, err := ex.Exec(ctx, taskOf("T-1061", tl1.StageActivateUser)); err == nil {
		t.Fatal("activate on empty sim must fail")
	}
	srv.DropConns()
	trace, err := ex.Exec(ctx, taskOf("T-1061", tl1.StagePreConfigOLT))
	if err != nil {
		t.Fatalf("Exec after drop: %v", err)
	}
	assertNoErrorResidue(t, trace)
	assertAligned(t, trace)
}

// ⑪ 最终失败保留可观测错误上下文:DENY 原始帧在案,EXEC-ERROR 收尾。
func TestExecTraceDenyKeepsErrorContext(t *testing.T) {
	_, m, _ := startSim(t, func(o *sim.Options) { o.DenyNext = "ADD-PONVLAN" })
	ex := newExec(m, testParams())
	trace, err := ex.Exec(context.Background(), taskOf("T-1062", tl1.StagePreConfigOLT))
	if err == nil {
		t.Fatal("want DENY failure")
	}
	if !strings.Contains(trace.Response, "DENY") {
		t.Fatalf("response missing DENY frame:\n%s", trace.Response)
	}
	if !strings.Contains(trace.Response, "EXEC-ERROR: ") {
		t.Fatalf("response missing EXEC-ERROR tail:\n%s", trace.Response)
	}
	if len(trace.Commands) != 4 {
		t.Fatalf("commands=%d want 4 (LST-ONU,ADD-ONU,LST-PONVLAN,ADD-PONVLAN):\n%s",
			len(trace.Commands), strings.Join(trace.Commands, "\n"))
	}
}

// ⑫ 重连重试后仍失败:被替代尝试以显式标头分隔保留(含断线错误文本),
// 最终错误 EXEC-ERROR 收尾,指令全量在案不静默丢弃。
func TestExecTraceReconnectFailKeepsSuperseded(t *testing.T) {
	srv, m, _ := startSim(t, func(o *sim.Options) { o.DenyNext = "ADD-PONVLAN" })
	ex := newExec(m, testParams())
	ctx := context.Background()
	// 暖会话且不消耗 DenyNext(ActivateUser 不发 ADD-PONVLAN)。
	if _, err := ex.Exec(ctx, taskOf("T-1063", tl1.StageActivateUser)); err == nil {
		t.Fatal("activate on empty sim must fail")
	}
	srv.DropConns()
	trace, err := ex.Exec(ctx, taskOf("T-1063", tl1.StagePreConfigOLT))
	if err == nil {
		t.Fatal("want failure after reconnect")
	}
	for _, want := range []string{
		"[attempt 1 superseded by reconnect]",
		"connection broken",
		"DENY",
		"EXEC-ERROR: ",
	} {
		if !strings.Contains(trace.Response, want) {
			t.Fatalf("response missing %q:\n%s", want, trace.Response)
		}
	}
	if len(trace.Commands) != 5 {
		t.Fatalf("commands=%d want 5 (attempt1 TRC 占位 1 条 + attempt2 4 条):\n%s",
			len(trace.Commands), strings.Join(trace.Commands, "\n"))
	}
}

// ⑬ 无设备交互的失败(unsupported stage):不留假应答,EXEC-ERROR 收尾。
func TestExecTraceUnsupportedStageErrorTail(t *testing.T) {
	_, m, _ := startSim(t, nil)
	ex := newExec(m, testParams())
	trace, err := ex.Exec(context.Background(), taskOf("T-1064", "unknownStage"))
	if err == nil {
		t.Fatal("want unsupported error")
	}
	if len(trace.Commands) != 0 || !strings.HasPrefix(trace.Response, "EXEC-ERROR: ") {
		t.Fatalf("trace commands=%d response=%q", len(trace.Commands), trace.Response)
	}
}
