package app

import (
	"context"
	"testing"

	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/config"
)

// TestSmsConfigResolverDevMode dev 环境(BOSS_DEV_MODE=true)强制日志通道:
// 即使 DB 配了阿里云 AK,解析结果也无凭据 → Dynamic 落 LogSender,不因模板缺失 500。
func TestSmsConfigResolverDevMode(t *testing.T) {
	cfg := config.Config{}
	cfg.Server.DevMode = true
	svc := &fakeParamLister{params: []user.Param{
		{Key: "sms.enabled", Value: "true"},
		{Key: "sms.accessKeyId", Value: "LTAI-test"},
		{Key: "sms.accessKeySecret", Value: "enc:v1:xxxx"},
	}}
	got, err := smsConfigResolver(svc, &cfg)(context.Background())
	if err != nil {
		t.Fatalf("resolver err: %v", err)
	}
	if !got.Enabled {
		t.Fatal("dev 模式应保持通道 enabled(走 LogSender),而非禁用")
	}
	if got.AccessKeyID != "" || got.AccessKeySecret != "" {
		t.Fatalf("dev 模式不应带真实凭据: AK=%q secret=%q", got.AccessKeyID, got.AccessKeySecret)
	}
}

// TestSmsConfigResolverProd 生产环境(BOSS_DEV_MODE=false)正常读 DB 凭据。
func TestSmsConfigResolverProd(t *testing.T) {
	cfg := config.Config{}
	svc := &fakeParamLister{params: []user.Param{
		{Key: "sms.accessKeyId", Value: "LTAI-prod"},
		{Key: "sms.contentCode.cn", Value: "SMS_123"},
	}}
	got, err := smsConfigResolver(svc, &cfg)(context.Background())
	if err != nil {
		t.Fatalf("resolver err: %v", err)
	}
	if got.AccessKeyID != "LTAI-prod" || got.ContentCodeCN != "SMS_123" {
		t.Fatalf("prod 应读 DB 配置: got %+v", got)
	}
}
