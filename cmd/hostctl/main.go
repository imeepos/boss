// hostctl: 102 宿主机 sidecar，接收 boss-server 的特权操作请求。
//
// 监听 127.0.0.1:39092（仅 loopback）。所有写操作需 HMAC-SHA256 签名：
//
//	X-Hostctl-Timestamp: <unix seconds>
//	X-Hostctl-Signature: hex(hmac_sha256(secret, "<ts>\n<method>\n<path>\n<body>"))
//
// 时间戳与服务器时间差 > 5min 拒绝，防重放。
//
// 路由：
//
//	GET  /healthz                              无鉴权
//	POST /v1/minio/rotate-secret               鉴权，写 root_password + 重启 minio
//
// 进程以非 root 运行（systemd User=imeepos），通过 sudoers 限定只能
// 执行 docker compose ... minio + 写 /etc/minio/secrets/*。
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	cfg = loadConfig()

	if _, err := os.Stat(cfg.SecretFile); err != nil {
		log.Fatalf("secret file missing: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealth)
	mux.Handle("POST /v1/minio/rotate-secret", auth(http.HandlerFunc(handleRotateSecret)))

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	log.Printf("hostctl listening on %s", cfg.Listen)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func loadConfig() *config {
	// 0.0.0.0 让 docker bridge 内的 boss-server 也能访问;
	// HMAC 鉴权保护写接口, healthz 无需认证。
	listen := flag.String("listen", "0.0.0.0:39093", "bind address")
	secret := flag.String("secret-file", "/etc/boss-host-ctl/hmac.key", "HMAC key file (0600)")
	cdir := flag.String("compose-dir", "/home/imeepos/boss/deployments", "compose 工作目录")
	cfile := flag.String("compose-file", "docker-compose.infra.yml,docker-compose.102.yml", "compose 文件，逗号分隔")
	project := flag.String("project", "boss-infra", "compose project 名")
	ctr := flag.String("minio-container", "boss-infra-minio-1", "minio 容器名")
	flag.Parse()
	return &config{
		Listen:      *listen,
		SecretFile:  *secret,
		ComposeDir:  *cdir,
		ComposeFile: *cfile,
		Project:     *project,
		MinioCtr:    *ctr,
	}
}
