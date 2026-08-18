package server

// Prometheus 指标:RED 基础(请求速率/错误/时延直方图)+ /metrics 端点(W11 可观测补挂)。

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "boss_http_requests_total", Help: "HTTP 请求计数",
	}, []string{"method", "path", "code"})
	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "boss_http_request_duration_seconds",
		Help:    "HTTP 时延分布",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5},
	}, []string{"method", "path"})
)

func init() {
	prometheus.MustRegister(httpRequests, httpDuration)
}

// metricsPath 归一化路由模板(避免高基数:参数折叠为 :param)。
func metricsPath(c *gin.Context) string {
	if c.FullPath() == "" {
		return "unmatched"
	}
	return c.FullPath()
}

// PrometheusMiddleware RED 指标中间件。
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" || c.Request.URL.Path == "/healthz" {
			c.Next() // 基础设施端点不进业务指标
			return
		}
		start := time.Now()
		c.Next()
		labels := prometheus.Labels{
			"method": c.Request.Method, "path": metricsPath(c),
			"code": strconv.Itoa(c.Writer.Status()),
		}
		httpRequests.With(labels).Inc()
		httpDuration.WithLabelValues(labels["method"], labels["path"]).Observe(time.Since(start).Seconds())
	}
}

// registerMetrics 暴露 /metrics。
func registerMetrics(r *gin.Engine) {
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
}
