package config

// AAA 域配置(阶段7 自研 RADIUS,性能服务群独立部署)。
// 自 config.go 平移为独立文件(零行为变更;避免 config.go 超 300 行红线);
// A5 增量:全局密钥兼容开关 + 厂商 VSA 限速属性名(字段权威 fields.md §8J)。

import "time"

// AAAConfig RADIUS/凭据/会话/per-NAS/VSA 配置。
type AAAConfig struct {
	AuthAddr           string        // RADIUS 认证端口(1812)
	AcctAddr           string        // RADIUS 计费端口(1813)
	Secret             string        // NAS 共享密钥(A5 起为兼容回退项,默认仅未注册 NAS 兜底用)
	AuthTTL            int           // 授权缓存 TTL 秒(默认 60)
	CDRTopic           string        // 话单 Kafka topic
	CredKey            string        // 凭据落库密钥材料(BOSS_AAA_CRED_KEY/文件;空=从 Secret 派生,生产应显式设置)
	AllowNoCred        bool          // BOSS_AAA_ALLOW_NO_CRED=true:未设密账号放行(迁移缓冲,默认关)
	LockThreshold      int           // BOSS_AAA_LOCK_THRESHOLD:连续失败锁定阈值(默认 5)
	LockWindow         time.Duration // BOSS_AAA_LOCK_WINDOW:锁定时长(默认 15m)
	SessionLimit       int           // 同一 LOID 并发会话上限(全局,默认 1;A2)
	CoAPort            int           // NAS CoA/DM 端口(RFC 5176,默认 3799;A2;per-NAS 注册表可覆盖)
	OfflineRetryMax    int           // Disconnect 不可达重试上限(默认 3;A2)
	ZombieAfter        time.Duration // 会话超时未更新判僵尸阈值(默认 2h;A2)
	GlobalSecretCompat bool          // BOSS_AAA_GLOBAL_SECRET_COMPAT=true:未注册 NAS 回退全局密钥(迁移缓冲,默认关)
	VSAHuawei          string        // BOSS_AAA_VSA_HUAWEI=上行属性名,下行属性名(默认 input-average-rate,output-average-rate)
	VSAZTE             string        // BOSS_AAA_VSA_ZTE=上行属性名,下行属性名(同默认对)
}

// loadAAA 从环境变量装载 AAA 配置。
func loadAAA(c *Config) {
	c.AAA.AuthAddr = getenv("BOSS_AAA_AUTH_ADDR", ":1812")
	c.AAA.AcctAddr = getenv("BOSS_AAA_ACCT_ADDR", ":1813")
	c.AAA.Secret = getenv("BOSS_AAA_SECRET", "boss-aaa-secret")
	c.AAA.AuthTTL = 60
	c.AAA.SessionLimit = getint("BOSS_AAA_SESSION_LIMIT", 1)
	c.AAA.CoAPort = getint("BOSS_AAA_COA_PORT", 3799)
	c.AAA.OfflineRetryMax = getint("BOSS_AAA_OFFLINE_RETRY_MAX", 3)
	c.AAA.ZombieAfter = getdur("BOSS_AAA_ZOMBIE_AFTER", 2*time.Hour)
	c.AAA.CDRTopic = getenv("BOSS_AAA_CDR_TOPIC", "boss-cdr")
	c.AAA.CredKey = readEnvOrFile("BOSS_AAA_CRED_KEY", "BOSS_AAA_CRED_KEY_FILE")
	c.AAA.AllowNoCred = getenv("BOSS_AAA_ALLOW_NO_CRED", "") == "true"
	c.AAA.LockThreshold = getint("BOSS_AAA_LOCK_THRESHOLD", 5)
	c.AAA.LockWindow = getdur("BOSS_AAA_LOCK_WINDOW", 15*time.Minute)
	c.AAA.GlobalSecretCompat = getenv("BOSS_AAA_GLOBAL_SECRET_COMPAT", "") == "true"
	c.AAA.VSAHuawei = getenv("BOSS_AAA_VSA_HUAWEI", "")
	c.AAA.VSAZTE = getenv("BOSS_AAA_VSA_ZTE", "")
}
