package config

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	Server struct {
		HTTPAddr string
		GRPCAddr string
	}
	Database struct{ DSN string }
	Redis    struct{ Addrs []string }
	Kafka    struct{ Brokers []string }
	MinIO    struct{ Endpoint, AccessKey, SecretKey string }
	Temporal struct{ Host string }

	// 102 扩展基础设施(见 deployments/docker-compose.102.extend.yml)
	APISIX        struct{ AdminURL string }
	Observability struct {
		PrometheusURL   string
		AlertmanagerURL string
		GrafanaURL      string
		LokiURL         string
		JaegerOTLP      string
	}
	OLAP  struct{ StarRocksDSN string }
	Flink struct{ Jobmanager string }

	// AAA(阶段7 自研 RADIUS,性能服务群独立部署)。
	AAA struct {
		AuthAddr string // RADIUS 认证端口(1812)
		AcctAddr string // RADIUS 计费端口(1813)
		Secret   string // NAS 共享密钥
		AuthTTL  int    // 授权缓存 TTL 秒(默认 60)
		CDRTopic string // 话单 Kafka topic
	}

	JWT struct {
		Secret string
		TTL    time.Duration
	}
}

// Load 从环境变量读取;文件/Nacos 热更新在阶段1迭代中接入。
func Load() *Config {
	c := &Config{}
	c.Server.HTTPAddr = getenv("BOSS_HTTP_ADDR", ":8080")
	c.Server.GRPCAddr = getenv("BOSS_GRPC_ADDR", ":9090")
	c.Database.DSN = getenv("BOSS_PG_DSN", "host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable")
	c.APISIX.AdminURL = getenv("BOSS_APISIX_ADMIN", "http://192.168.0.102:29180")
	c.Observability.PrometheusURL = getenv("BOSS_PROMETHEUS_URL", "http://192.168.0.102:19090")
	c.Observability.AlertmanagerURL = getenv("BOSS_ALERTMANAGER_URL", "http://192.168.0.102:19093")
	c.Observability.GrafanaURL = getenv("BOSS_GRAFANA_URL", "http://192.168.0.102:19300")
	c.Observability.LokiURL = getenv("BOSS_LOKI_URL", "http://192.168.0.102:19310")
	c.Observability.JaegerOTLP = getenv("BOSS_JAEGER_OTLP", "192.168.0.102:16831")
	c.OLAP.StarRocksDSN = getenv("BOSS_STARROCKS_DSN", "root@tcp(192.168.0.102:29030)/")
	c.Flink.Jobmanager = getenv("BOSS_FLINK_JM", "192.168.0.102:18082")
	c.AAA.AuthAddr = getenv("BOSS_AAA_AUTH_ADDR", ":1812")
	c.AAA.AcctAddr = getenv("BOSS_AAA_ACCT_ADDR", ":1813")
	c.AAA.Secret = getenv("BOSS_AAA_SECRET", "boss-aaa-secret")
	c.AAA.AuthTTL = 60
	c.Kafka.Brokers = getlist("BOSS_KAFKA_BROKERS", []string{"192.168.0.102:29092"})
	c.AAA.CDRTopic = getenv("BOSS_AAA_CDR_TOPIC", "boss-cdr")
	c.JWT.Secret = getenv("BOSS_JWT_SECRET", "change-me")
	c.JWT.TTL = 24 * time.Hour
	return c
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getlist(k string, def []string) []string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	out := []string{}
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
