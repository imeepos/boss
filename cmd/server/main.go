// server 业务单体入口:阶段1-6 全部业务域 + 阶段7 接口预留。
package main

import (
	"log"

	"github.com/ymm-001/boss/internal/app"
)

func main() {
	a, err := app.New()
	if err != nil {
		log.Fatalf("wiring: %v", err)
	}
	_ = a
	log.Println("boss-server: skeleton ready, waiting for stage-1 implementation")
}
