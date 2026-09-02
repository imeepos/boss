// tl1sim 是 U2000 北向 TL1 接口仿真器命令行包装;测试基建,不进生产部署。
// 行为细节见 cmd/tl1sim/sim 包。
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ymm-001/boss/cmd/tl1sim/sim"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:13027", "listen address")
	user := flag.String("user", "admin", "LOGIN UN")
	pass := flag.String("pass", "admin", "LOGIN PWD")
	oper := flag.String("onu-oper-state", "UP", "new ONU OperState: UP|Power-Off|LOS")
	delay := flag.Int("delay-ms", 0, "reply one DELAY frame and wait this long before the final response")
	denyNext := flag.String("deny-next", "", "deny the next command of this verb once, e.g. ADD-ONU")
	recordPath := flag.String("record", "", "JSONL file to record received commands")
	flag.Parse()

	srv, err := sim.New(sim.Options{
		Addr:         *addr,
		User:         *user,
		Pass:         *pass,
		ONUOperState: *oper,
		DelayMs:      *delay,
		DenyNext:     *denyNext,
		RecordPath:   *recordPath,
	})
	if err != nil {
		log.Fatalf("[tl1sim] START FAILED: %v", err)
	}
	srv.Start()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	if err := srv.Stop(); err != nil {
		log.Printf("[tl1sim] STOP FAILED: %v", err)
	}
}
