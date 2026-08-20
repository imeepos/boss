// Package server gin/grpc server 组装、健康检查、优雅退出。
// 阶段1 起提供最小可运行的 HTTP 骨架:健康检查 + 优雅退出 + trace 注入(见 README 阶段映射)。
package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/pkg/middleware"
)

// Config 服务器装配入参(由 internal/pkg/config 展开后传入)。
type Config struct {
	HTTPAddr    string
	CORSOrigins []string
}

// New 构建一个具备健康检查与 recover 的 gin engine。
// 健康检查端点 /healthz 供 K8s liveness/readiness 探活;业务路由由 app 层在此 engine 上注册。
func New(cfg Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.TraceID(), middleware.AccessLog(), PrometheusMiddleware(), cors(cfg.CORSOrigins))
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	registerMetrics(r)
	return r
}

func cors(origins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if !originAllowed(origin, allowed) {
			c.Next()
			return
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// originAllowed 精确白名单命中即放行;开发本机源(localhost/127.0.0.1)不限端口——
// vite 端口随占用漂移(5173→5175…),逐个枚举不可维护。
func originAllowed(origin string, allowed map[string]struct{}) bool {
	if _, ok := allowed[origin]; ok {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// Run 启动 HTTP 并监听 OS 信号实现优雅退出。
// 返回 error 仅用于启动失败的致命场景;优雅退出路径返回 nil。
func Run(r *gin.Engine, addr string) error {
	srv := &http.Server{Addr: addr, Handler: r}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("boss-server http listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		log.Printf("received %s, shutting down gracefully", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}
