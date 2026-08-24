package server

// Prometheus 指标:RED 基础(请求速率/错误/时延直方图)+ /metrics 端点(W11 可观测补挂)。

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
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

// 连接池运行指标(pgxpool.Stat 快照,gauge):SLO 基线 W11 要求观察池等待尾延迟与连接使用率。
var (
	poolDesc = map[string]*prometheus.Desc{
		"total":    prometheus.NewDesc("boss_db_pool_total_conns", "连接池总连接数", nil, nil),
		"idle":     prometheus.NewDesc("boss_db_pool_idle_conns", "连接池空闲连接数", nil, nil),
		"acquired": prometheus.NewDesc("boss_db_pool_acquired_conns", "连接池在用连接数", nil, nil),
		"max":      prometheus.NewDesc("boss_db_pool_max_conns", "连接池最大连接数", nil, nil),
		"wait":     prometheus.NewDesc("boss_db_pool_acquire_seconds", "获取连接平均等待秒数(池空排队)", nil, nil),
	}
)

type poolCollector struct{ pool *pgxpool.Pool }

// Describe 声明全部连接池指标。
func (c *poolCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range poolDesc {
		ch <- d
	}
}

// Collect 采集 pgxpool.Stat 当前快照。
func (c *poolCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.pool.Stat()
	ch <- prometheus.MustNewConstMetric(poolDesc["total"], prometheus.GaugeValue, float64(s.TotalConns()))
	ch <- prometheus.MustNewConstMetric(poolDesc["idle"], prometheus.GaugeValue, float64(s.IdleConns()))
	ch <- prometheus.MustNewConstMetric(poolDesc["acquired"], prometheus.GaugeValue, float64(s.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(poolDesc["max"], prometheus.GaugeValue, float64(s.MaxConns()))
	wait := 0.0
	if s.AcquireCount() > 0 {
		wait = s.AcquireDuration().Seconds() / float64(s.AcquireCount())
	}
	ch <- prometheus.MustNewConstMetric(poolDesc["wait"], prometheus.GaugeValue, wait)
}

// RegisterPoolMetrics 注册连接池指标采集(幂等,重复注册 panic 由调用方保证单次)。
func RegisterPoolMetrics(pool *pgxpool.Pool) {
	prometheus.MustRegister(&poolCollector{pool: pool})
}
