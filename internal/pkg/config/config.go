package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server struct {
		HTTPAddr string
		GRPCAddr string
		DevMode  bool // BOSS_DEV_MODE=true:注册开发专用调试端点(回显验证码等);生产严禁开启。
	}
	CORS struct {
		Origins []string
	}
	Database struct{ DSN string }
	Redis    struct{ Addrs []string }
	Kafka    struct{ Brokers []string }
	Events   struct{ Topic string } // 状态变更事件 topic
	MinIO struct {
		Endpoint, AccessKey, SecretKey, Bucket string
		UseSSL                                 bool
	}
	HostCtl struct {
		URL    string // http://172.26.0.1:39093 (docker bridge gateway)
		HMACKey string // 预共享 HMAC 密钥
	}
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

	// Bootstrap 启动引导超管(SYS 域):AdminPass 非空才启用,空则跳过。
	Bootstrap struct {
		AdminUser string
		AdminPass string
	}

	// Provisioner(阶段7 下发守护进程,债务偿还:真实 Telnet 执行器)。
	Provisioner struct {
		OLTAddr  string // OLT 管理地址 host:port
		OLTUser  string
		OLTPass  string
		Interval time.Duration
	}
	// Collector(阶段7 采集器,债务偿还:真实 SNMP 采集源)。
	Collector struct {
		Targets       []string // "code@host:port,..."
		Community     string
		OpticalOID    string
		PacketLossOID string
		StatusOID     string
		Interval      time.Duration
	}
	// Analytics(阶段9 五大指标口径单价,可解释公式系数)。
	Analytics struct {
		MaintUnitCost float64 // 单次维护成本(元)
		PortUnitCost  float64 // 单端口扩容成本(元)
		Backend       string  // pg | starrocks(OLAP 宽表)
	}
	// Report(阶段9 自动报告)。
	Report struct {
		Period    string        // daily/weekly/monthly/quarterly
		Interval  time.Duration // 生成巡检周期
		PushTopic string        // 快照推送 Kafka topic
	}
	// SMS 验证码短信通道(阿里云国际短信;凭据为空时降级日志通道)。
	SMS struct {
		AccessKeyID     string
		AccessKeySecret string
		From            string // 阿里云国际 SenderID
	}
	// RealID 实名二要素核验通道(阿里云实人认证 Id2MetaVerify;凭据为空时通道不注册,落 PENDING 走人工核验)。
	RealID struct {
		AccessKeyID     string
		AccessKeySecret string
	}
	// Stripe 支付通道(卡收单;APIKey 为空时通道不注册,缴费走既有模拟直落账)。
	Stripe struct {
		APIKey     string // sk_... 密钥(引用不存值)
		WebhookSec string // endpoint signing secret(whsec_...)
		Currency   string // 记账币种小写(如 php)
		APIBaseURL string // 覆盖 API 地址(测试/代理用,空=官方)
	}
}

// Load 从环境变量读取;文件/Nacos 热更新在阶段1迭代中接入。
func Load() *Config {
	c := &Config{}
	c.Server.HTTPAddr = getenv("BOSS_HTTP_ADDR", ":8080")
	c.Server.GRPCAddr = getenv("BOSS_GRPC_ADDR", ":9090")
	c.Server.DevMode = getenv("BOSS_DEV_MODE", "") == "true"
	c.CORS.Origins = getlist("BOSS_CORS_ORIGINS", []string{"http://localhost:5173", "http://localhost:5174"})
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
	c.Events.Topic = getenv("BOSS_EVENTS_TOPIC", "boss-order-events")
	c.JWT.Secret = getenv("BOSS_JWT_SECRET", "change-me")
	c.JWT.TTL = getdur("BOSS_JWT_TTL", 7*24*time.Hour)
	c.Bootstrap.AdminUser = getenv("BOSS_ADMIN_USERNAME", "admin")
	c.Bootstrap.AdminPass = getenv("BOSS_ADMIN_PASSWORD", "")

	c.Provisioner.OLTAddr = getenv("BOSS_PROVISION_OLT_ADDR", "0")
	c.Provisioner.OLTUser = getenv("BOSS_PROVISION_OLT_USER", "admin")
	c.Provisioner.OLTPass = getenv("BOSS_PROVISION_OLT_PASS", "admin")
	c.Provisioner.Interval = 5 * time.Second

	c.Collector.Targets = getlist("BOSS_SNMP_TARGETS", nil)
	c.Collector.Community = getenv("BOSS_SNMP_COMMUNITY", "public")
	c.Collector.OpticalOID = getenv("BOSS_SNMP_OPTICAL_OID", "1.3.6.1.4.1.100.1")
	c.Collector.PacketLossOID = getenv("BOSS_SNMP_PACKETLOSS_OID", "1.3.6.1.4.1.100.2")
	c.Collector.StatusOID = getenv("BOSS_SNMP_STATUS_OID", "1.3.6.1.2.1.2.2.1.8")
	c.Collector.Interval = 30 * time.Second

	c.Analytics.MaintUnitCost = getfloat("BOSS_MAINT_UNIT_COST", 50)
	c.Analytics.PortUnitCost = getfloat("BOSS_PORT_UNIT_COST", 800)
	c.Analytics.Backend = getenv("BOSS_ANALYTICS_BACKEND", "pg")

	c.Report.Period = getenv("BOSS_REPORT_PERIOD", "daily")
	c.Report.PushTopic = getenv("BOSS_REPORT_PUSH_TOPIC", "boss-report-snapshots")
	c.Report.Interval = 6 * time.Hour

	c.SMS.AccessKeyID = getenv("BOSS_SMS_ALIYUN_AK_ID", "")
	c.SMS.AccessKeySecret = getenv("BOSS_SMS_ALIYUN_AK_SECRET", "")
	c.SMS.From = getenv("BOSS_SMS_ALIYUN_FROM", "")

	c.RealID.AccessKeyID = getenv("BOSS_REALID_ALIYUN_AK_ID", "")
	c.RealID.AccessKeySecret = getenv("BOSS_REALID_ALIYUN_AK_SECRET", "")

	c.Stripe.APIKey = getenv("BOSS_STRIPE_API_KEY", "")
	c.Stripe.WebhookSec = getenv("BOSS_STRIPE_WEBHOOK_SECRET", "")
	c.Stripe.Currency = getenv("BOSS_STRIPE_CURRENCY", "php")
	c.Stripe.APIBaseURL = getenv("BOSS_STRIPE_API_BASE", "")

	c.MinIO.Endpoint = getenv("BOSS_MINIO_ENDPOINT", "192.168.0.102:29000")
	c.MinIO.AccessKey = getenv("BOSS_MINIO_ACCESS_KEY", "boss")
	c.MinIO.SecretKey = getenv("BOSS_MINIO_SECRET_KEY", "boss12345")
	c.MinIO.Bucket = getenv("BOSS_MINIO_BUCKET", "boss-attachments")
	c.MinIO.UseSSL = getenv("BOSS_MINIO_USE_SSL", "false") == "true"
	c.HostCtl.URL = getenv("BOSS_HOSTCTL_URL", "http://172.26.0.1:39093")
	c.HostCtl.HMACKey = readEnvOrFile("BOSS_HOSTCTL_HMAC_KEY", "BOSS_HOSTCTL_HMAC_KEY_FILE")
	return c
}

// getdur 环境变量取时长(如 720h/30m),缺省 d。
func getdur(k string, d time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if t, err := time.ParseDuration(v); err == nil && t > 0 {
			return t
		}
	}
	return d
}

// getfloat 环境变量取浮点,缺省 d。
func getfloat(k string, d float64) float64 {
	if v := os.Getenv(k); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return d
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

// readEnvOrFile 先读 envKey 环境变量,若为空则读 fileKey 对应的文件路径内容。
// 用于密钥类配置:env 直传优先,文件挂载兜底(避免明文进 compose/git)。
func readEnvOrFile(envKey, fileKey string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if fp := os.Getenv(fileKey); fp != "" {
		data, err := os.ReadFile(fp)
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return ""
}
