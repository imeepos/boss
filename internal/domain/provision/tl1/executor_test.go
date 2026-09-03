package tl1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ymm-001/boss/cmd/tl1sim/sim"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/domain/provision/tl1"
)

// ---- 进程内 e2e 基建:仿真器 + Manager + 留痕断言 ----

// startSim 起仿真器并接 Manager;返回实例与留痕文件路径。
func startSim(t *testing.T, mod func(*sim.Options)) (*sim.Server, *tl1.Manager, string) {
	t.Helper()
	opt := sim.Options{
		Addr:       "127.0.0.1:0",
		User:       "admin",
		Pass:       "admin",
		RecordPath: filepath.Join(t.TempDir(), "tl1sim.jsonl"),
	}
	if mod != nil {
		mod(&opt)
	}
	srv, err := sim.New(opt)
	if err != nil {
		t.Fatalf("sim.New: %v", err)
	}
	srv.Start()
	t.Cleanup(func() { _ = srv.Stop() })
	m := tl1.NewManager(tl1.Config{
		Addr: srv.Addr(), User: opt.User, Pass: opt.Pass,
		Keepalive: time.Minute, CmdTimeout: 5 * time.Second, DialTimeout: 2 * time.Second,
	})
	return srv, m, opt.RecordPath
}

// testParams 一单两业务:Internet 双层 + TR069 单层(不带 SVLAN)。
func testParams() tl1.Params {
	svc := func(name string, hasSVLAN bool) tl1.PONVLANParams {
		return tl1.PONVLANParams{
			Name: name, OLTID: "10.0.0.9", PONID: "NA-0-7-5",
			ONUIDType: "LOID", ONUID: "loid-0001",
			SVLAN: 1000, CVLAN: 100, UV: 100, SCOS: 5, HasSVLAN: hasSVLAN,
		}
	}
	return tl1.Params{
		OLTID: "10.0.0.9", PONID: "NA-0-7-5",
		AuthType: "LOID", ONUID: "loid-0001", ONUNo: 7,
		ONUType: "FTTH_E8C", Desc: "PRV-ORD-0001",
		Services: []tl1.PONVLANParams{svc("Internet", true), svc("TR069", false)},
	}
}

// fakeResolver 进程内注入的参数解析桩。
type fakeResolver struct{ p tl1.Params }

func (f fakeResolver) Resolve(ctx context.Context, t provision.Task) (tl1.Params, error) {
	return f.p, nil
}

func newExec(m *tl1.Manager, p tl1.Params) *tl1.Executor {
	p.Endpoint = m.Endpoint()
	return tl1.NewExecutor(fakeResolver{p: p})
}

func taskOf(no, stage string) provision.Task {
	return provision.Task{TaskNo: no, StageEvent: stage}
}

// recordVerbs 读留痕 JSONL,返回指令动词序列。
func recordVerbs(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	var verbs []string
	for _, ln := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if ln == "" {
			continue
		}
		var o struct {
			Verb string
		}
		if err := json.Unmarshal([]byte(ln), &o); err != nil {
			t.Fatalf("bad record line %q: %v", ln, err)
		}
		verbs = append(verbs, o.Verb)
	}
	return verbs
}

func countVerbs(verbs []string, verb string) int {
	n := 0
	for _, v := range verbs {
		if v == verb {
			n++
		}
	}
	return n
}

// captureLogs 收编全局 log 执行 fn,返回日志文本;测试串行,不并行使用。
func captureLogs(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	out := log.Writer()
	flags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(out)
		log.SetFlags(flags)
	}()
	fn()
	return buf.String()
}

func assertCounts(t *testing.T, verbs []string, onu, ponvlan int) {
	t.Helper()
	if got := countVerbs(verbs, "ADD-ONU"); got != onu {
		t.Fatalf("ADD-ONU=%d want %d", got, onu)
	}
	if got := countVerbs(verbs, "ADD-PONVLAN"); got != ponvlan {
		t.Fatalf("ADD-PONVLAN=%d want %d", got, ponvlan)
	}
}

// ① 环节7 首次全流程:ADD-ONU×1 + ADD-PONVLAN×2(record 断言)。
func TestExecPreConfigFirstRun(t *testing.T) {
	_, m, rec := startSim(t, nil)
	ex := newExec(m, testParams())
	trace, err := ex.Exec(context.Background(), taskOf("T-1001", tl1.StagePreConfigOLT))
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if trace.Driver != provision.DriverTL1 {
		t.Fatalf("driver=%q, want tl1", trace.Driver)
	}
	assertCounts(t, recordVerbs(t, rec), 1, 2)
}

// ② 环节7 重跑幂等:零新增写指令,日志含 ONU/PONVLAN 两条 SKIP。
func TestExecPreConfigRerunIdempotent(t *testing.T) {
	_, m, rec := startSim(t, nil)
	ex := newExec(m, testParams())
	task := taskOf("T-1002", tl1.StagePreConfigOLT)
	ctx := context.Background()
	if _, err := ex.Exec(ctx, task); err != nil {
		t.Fatalf("first Exec: %v", err)
	}
	var logs string
	var err error
	logs = captureLogs(t, func() {
		_, err = ex.Exec(ctx, task)
	})
	if err != nil {
		t.Fatalf("rerun Exec: %v", err)
	}
	assertCounts(t, recordVerbs(t, rec), 1, 2)
	for _, want := range []string{"IDEMPOTENT SKIP(ONU-EXISTS)", "IDEMPOTENT SKIP(PONVLAN-EXISTS)"} {
		if !strings.Contains(logs, want) {
			t.Fatalf("logs missing %q: %s", want, logs)
		}
	}
	if strings.Contains(logs, "EXEC FAILED") {
		t.Fatalf("unexpected EXEC FAILED: %s", logs)
	}
}

// ③ 环节7 中途 DENY:整体 error 带 EN/ENDESC/ctag,EXEC FAILED 留痕,后续写不再发生。
func TestExecPreConfigDenyMidway(t *testing.T) {
	_, m, rec := startSim(t, func(o *sim.Options) { o.DenyNext = "ADD-PONVLAN" })
	ex := newExec(m, testParams())
	var err error
	logs := captureLogs(t, func() {
		_, err = ex.Exec(context.Background(), taskOf("T-1003", tl1.StagePreConfigOLT))
	})
	if err == nil {
		t.Fatal("want error on DENY")
	}
	for _, want := range []string{"EN=", "ENDESC=", "ctag=", "ADD-PONVLAN"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("err %q missing %q", err, want)
		}
	}
	if !strings.Contains(logs, "EXEC FAILED") {
		t.Fatalf("logs missing EXEC FAILED: %s", logs)
	}
	// 留痕记"收到"而非"执行成功":被 DENY 的首条 ADD-PONVLAN 在案,第二条未发出。
	assertCounts(t, recordVerbs(t, rec), 1, 1)
}

// ⑥ DELAY 注入:先收 DELAY 帧仍拿最终响应,全流程成功。
func TestExecDelayStillSucceeds(t *testing.T) {
	_, m, rec := startSim(t, func(o *sim.Options) { o.DelayMs = 60 })
	ex := newExec(m, testParams())
	if _, err := ex.Exec(context.Background(), taskOf("T-1006", tl1.StagePreConfigOLT)); err != nil {
		t.Fatalf("Exec: %v", err)
	}
	assertCounts(t, recordVerbs(t, rec), 1, 2)
}
