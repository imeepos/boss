package config

// Load 单测:默认值 + 环境变量覆盖 + 解析分支(非法时长/浮点回退缺省、列表裁剪空白)。

import (
	"testing"
	"time"
)

// setenvs 批量注入环境变量,测试结束自动还原。
func setenvs(t *testing.T, kv map[string]string) {
	t.Helper()
	for k := range kv {
		t.Setenv(k, kv[k])
	}
}

func TestLoadDefaults(t *testing.T) {
	c := Load()
	if c.Server.HTTPAddr != ":8080" || c.Server.GRPCAddr != ":9090" {
		t.Fatalf("addr defaults: %+v", c.Server)
	}
	if len(c.CORS.Origins) != 2 || c.CORS.Origins[0] != "http://localhost:5173" {
		t.Fatalf("cors defaults: %v", c.CORS.Origins)
	}
	if c.JWT.Secret != "change-me" || c.JWT.TTL != 7*24*time.Hour {
		t.Fatalf("jwt defaults: %+v", c.JWT)
	}
	if c.Bootstrap.AdminUser != "admin" || c.Bootstrap.AdminPass != "" {
		t.Fatalf("bootstrap defaults: %+v", c.Bootstrap)
	}
	if c.AAA.AuthTTL != 60 || c.AAA.AuthAddr != ":1812" || c.AAA.AcctAddr != ":1813" || c.AAA.Secret != "boss-aaa-secret" || c.AAA.CDRTopic != "boss-cdr" {
		t.Fatalf("aaa defaults: %+v", c.AAA)
	}
	if c.Provisioner.Interval != 5*time.Second || c.Collector.Interval != 30*time.Second || c.Report.Interval != 6*time.Hour {
		t.Fatalf("interval defaults: %+v %+v %+v", c.Provisioner, c.Collector, c.Report)
	}
	if c.Collector.Targets != nil {
		t.Fatalf("snmp targets default want nil, got %v", c.Collector.Targets)
	}
	if c.Analytics.MaintUnitCost != 50 || c.Analytics.PortUnitCost != 800 || c.Analytics.Backend != "pg" {
		t.Fatalf("analytics defaults: %+v", c.Analytics)
	}
	if c.MinIO.UseSSL {
		t.Fatal("minio UseSSL default want false")
	}
	if c.Stripe.Currency != "php" || c.Stripe.APIKey != "" || c.Stripe.APIBaseURL != "" {
		t.Fatalf("stripe defaults: %+v", c.Stripe)
	}
	if c.SMS.AccessKeyID != "" || c.RealID.AccessKeyID != "" {
		t.Fatalf("sms/realid defaults: %+v %+v", c.SMS, c.RealID)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	setenvs(t, map[string]string{
		"BOSS_HTTP_ADDR":               ":9000",
		"BOSS_GRPC_ADDR":               ":9001",
		"BOSS_CORS_ORIGINS":            " http://a.com , ,http://b.com,",
		"BOSS_PG_DSN":                  "dsn-x",
		"BOSS_APISIX_ADMIN":            "http://apisix",
		"BOSS_PROMETHEUS_URL":          "http://prom",
		"BOSS_ALERTMANAGER_URL":        "http://am",
		"BOSS_GRAFANA_URL":             "http://grafana",
		"BOSS_LOKI_URL":                "http://loki",
		"BOSS_JAEGER_OTLP":             "jaeger:4317",
		"BOSS_STARROCKS_DSN":           "sr-dsn",
		"BOSS_FLINK_JM":                "flink:8081",
		"BOSS_AAA_AUTH_ADDR":           ":2812",
		"BOSS_AAA_ACCT_ADDR":           ":2813",
		"BOSS_AAA_SECRET":              "s3cr3t",
		"BOSS_AAA_CDR_TOPIC":           "cdr-t",
		"BOSS_KAFKA_BROKERS":           "k1:9092, k2:9092",
		"BOSS_EVENTS_TOPIC":            "evt-t",
		"BOSS_JWT_SECRET":              "jwt-s",
		"BOSS_JWT_TTL":                 "720h",
		"BOSS_ADMIN_USERNAME":          "root",
		"BOSS_ADMIN_PASSWORD":          "pass",
		"BOSS_PROVISION_OLT_ADDR":      "1.2.3.4:23",
		"BOSS_PROVISION_OLT_USER":      "ou",
		"BOSS_PROVISION_OLT_PASS":      "op",
		"BOSS_SNMP_TARGETS":            "c1@10.0.0.1:161,c2@10.0.0.2:161",
		"BOSS_SNMP_COMMUNITY":          "private",
		"BOSS_SNMP_OPTICAL_OID":        "1.1",
		"BOSS_SNMP_PACKETLOSS_OID":     "1.2",
		"BOSS_SNMP_STATUS_OID":         "1.3",
		"BOSS_MAINT_UNIT_COST":         "12.5",
		"BOSS_PORT_UNIT_COST":          "900",
		"BOSS_ANALYTICS_BACKEND":       "starrocks",
		"BOSS_REPORT_PERIOD":           "weekly",
		"BOSS_REPORT_PUSH_TOPIC":       "push-t",
		"BOSS_SMS_ALIYUN_AK_ID":        "sms-id",
		"BOSS_SMS_ALIYUN_AK_SECRET":    "sms-sec",
		"BOSS_REALID_ALIYUN_AK_ID":     "rid-id",
		"BOSS_REALID_ALIYUN_AK_SECRET": "rid-sec",
		"BOSS_STRIPE_API_KEY":          "sk_x",
		"BOSS_STRIPE_WEBHOOK_SECRET":   "whsec_x",
		"BOSS_STRIPE_CURRENCY":         "usd",
		"BOSS_STRIPE_API_BASE":         "http://stripe.test",
		"BOSS_MINIO_ENDPOINT":          "minio:9000",
		"BOSS_MINIO_ACCESS_KEY":        "ak",
		"BOSS_MINIO_SECRET_KEY":        "sk",
		"BOSS_MINIO_BUCKET":            "bkt",
		"BOSS_MINIO_USE_SSL":           "true",
	})

	c := Load()
	if c.Server.HTTPAddr != ":9000" || c.Server.GRPCAddr != ":9001" {
		t.Fatalf("addr: %+v", c.Server)
	}
	if got := c.CORS.Origins; len(got) != 2 || got[0] != "http://a.com" || got[1] != "http://b.com" {
		t.Fatalf("cors: %v", got)
	}
	if c.Database.DSN != "dsn-x" || c.APISIX.AdminURL != "http://apisix" {
		t.Fatalf("db/apisix: %+v %+v", c.Database, c.APISIX)
	}
	if c.Observability.PrometheusURL != "http://prom" || c.Observability.AlertmanagerURL != "http://am" ||
		c.Observability.GrafanaURL != "http://grafana" || c.Observability.LokiURL != "http://loki" ||
		c.Observability.JaegerOTLP != "jaeger:4317" {
		t.Fatalf("obs: %+v", c.Observability)
	}
	if c.OLAP.StarRocksDSN != "sr-dsn" || c.Flink.Jobmanager != "flink:8081" {
		t.Fatalf("olap/flink: %+v %+v", c.OLAP, c.Flink)
	}
	if c.AAA.AuthAddr != ":2812" || c.AAA.AcctAddr != ":2813" || c.AAA.Secret != "s3cr3t" || c.AAA.CDRTopic != "cdr-t" {
		t.Fatalf("aaa: %+v", c.AAA)
	}
	if got := c.Kafka.Brokers; len(got) != 2 || got[0] != "k1:9092" || got[1] != "k2:9092" {
		t.Fatalf("kafka: %v", got)
	}
	if c.Events.Topic != "evt-t" || c.JWT.Secret != "jwt-s" || c.JWT.TTL != 720*time.Hour {
		t.Fatalf("events/jwt: %+v %+v", c.Events, c.JWT)
	}
	if c.Bootstrap.AdminUser != "root" || c.Bootstrap.AdminPass != "pass" {
		t.Fatalf("bootstrap: %+v", c.Bootstrap)
	}
	if c.Provisioner.OLTAddr != "1.2.3.4:23" || c.Provisioner.OLTUser != "ou" || c.Provisioner.OLTPass != "op" {
		t.Fatalf("provisioner: %+v", c.Provisioner)
	}
	if got := c.Collector.Targets; len(got) != 2 || got[0] != "c1@10.0.0.1:161" {
		t.Fatalf("snmp targets: %v", got)
	}
	if c.Collector.Community != "private" || c.Collector.OpticalOID != "1.1" || c.Collector.PacketLossOID != "1.2" || c.Collector.StatusOID != "1.3" {
		t.Fatalf("snmp: %+v", c.Collector)
	}
	if c.Analytics.MaintUnitCost != 12.5 || c.Analytics.PortUnitCost != 900 || c.Analytics.Backend != "starrocks" {
		t.Fatalf("analytics: %+v", c.Analytics)
	}
	if c.Report.Period != "weekly" || c.Report.PushTopic != "push-t" {
		t.Fatalf("report: %+v", c.Report)
	}
	if c.SMS.AccessKeyID != "sms-id" || c.SMS.AccessKeySecret != "sms-sec" {
		t.Fatalf("sms: %+v", c.SMS)
	}
	if c.RealID.AccessKeyID != "rid-id" || c.RealID.AccessKeySecret != "rid-sec" {
		t.Fatalf("realid: %+v", c.RealID)
	}
	if c.Stripe.APIKey != "sk_x" || c.Stripe.WebhookSec != "whsec_x" || c.Stripe.Currency != "usd" || c.Stripe.APIBaseURL != "http://stripe.test" {
		t.Fatalf("stripe: %+v", c.Stripe)
	}
	if c.MinIO.Endpoint != "minio:9000" || c.MinIO.AccessKey != "ak" || c.MinIO.SecretKey != "sk" || c.MinIO.Bucket != "bkt" || !c.MinIO.UseSSL {
		t.Fatalf("minio: %+v", c.MinIO)
	}
}

func TestParseFallbacks(t *testing.T) {
	setenvs(t, map[string]string{
		"BOSS_JWT_TTL":         "abc", // 非法时长
		"BOSS_MAINT_UNIT_COST": "-x",  // 非法浮点
		"BOSS_PORT_UNIT_COST":  "0",   // 合法 0
		"BOSS_CORS_ORIGINS":    ",",   // 全空白项回退缺省? getlist 非空输入返回空列表
	})

	t.Run("非法时长回退缺省", func(t *testing.T) {
		if got := getdur("BOSS_JWT_TTL", time.Minute); got != time.Minute {
			t.Fatalf("getdur=%v, want 1m", got)
		}
	})
	t.Run("非正时长回退缺省", func(t *testing.T) {
		t.Setenv("BOSS_JWT_TTL", "-5m")
		if got := getdur("BOSS_JWT_TTL", time.Minute); got != time.Minute {
			t.Fatalf("getdur=%v, want 1m", got)
		}
	})
	t.Run("空时长回退缺省", func(t *testing.T) {
		t.Setenv("BOSS_JWT_TTL", "")
		if got := getdur("BOSS_JWT_TTL", time.Minute); got != time.Minute {
			t.Fatalf("getdur=%v, want 1m", got)
		}
	})
	t.Run("非法浮点回退缺省", func(t *testing.T) {
		if got := getfloat("BOSS_MAINT_UNIT_COST", 1.5); got != 1.5 {
			t.Fatalf("getfloat=%v, want 1.5", got)
		}
	})
	t.Run("空浮点回退缺省", func(t *testing.T) {
		t.Setenv("BOSS_MAINT_UNIT_COST", "")
		if got := getfloat("BOSS_MAINT_UNIT_COST", 2.5); got != 2.5 {
			t.Fatalf("getfloat=%v, want 2.5", got)
		}
	})
	t.Run("合法零值浮点生效", func(t *testing.T) {
		if got := getfloat("BOSS_PORT_UNIT_COST", 9); got != 0 {
			t.Fatalf("getfloat=%v, want 0", got)
		}
	})
	t.Run("getenv空值回退", func(t *testing.T) {
		t.Setenv("BOSS_K", "")
		if got := getenv("BOSS_K", "def"); got != "def" {
			t.Fatalf("getenv=%q, want def", got)
		}
		t.Setenv("BOSS_K", "v")
		if got := getenv("BOSS_K", "def"); got != "v" {
			t.Fatalf("getenv=%q, want v", got)
		}
	})
	t.Run("getlist空输入回退缺省", func(t *testing.T) {
		if got := getlist("BOSS_LIST_NONE", []string{"a"}); len(got) != 1 || got[0] != "a" {
			t.Fatalf("getlist=%v", got)
		}
	})
	t.Run("getlist全空白返回空列表", func(t *testing.T) {
		t.Setenv("BOSS_CORS_ORIGINS", " , ,")
		if got := getlist("BOSS_CORS_ORIGINS", nil); len(got) != 0 {
			t.Fatalf("getlist=%v, want empty", got)
		}
	})
}
