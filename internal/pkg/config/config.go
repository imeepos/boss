package config

import (
	"os"
	"time"
)

type Config struct {
	Server struct {
		HTTPAddr string
		GRPCAddr string
	}
	Database struct{ DSN string }
	Redis     struct{ Addrs []string }
	Kafka     struct{ Brokers []string }
	MinIO     struct{ Endpoint, AccessKey, SecretKey string }
	Temporal  struct{ Host string }
	JWT       struct {
		Secret string
		TTL    time.Duration
	}
}

// Load 从环境变量读取;文件/Nacos 热更新在阶段1迭代中接入。
func Load() *Config {
	c := &Config{}
	c.Server.HTTPAddr = getenv("BOSS_HTTP_ADDR", ":8080")
	c.Server.GRPCAddr = getenv("BOSS_GRPC_ADDR", ":9090")
	c.Database.DSN = getenv("BOSS_PG_DSN", "host=localhost port=5432 user=boss password=boss dbname=boss sslmode=disable")
	c.JWT.Secret = getenv("BOSS_JWT_SECRET", "change-me")
	return c
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
