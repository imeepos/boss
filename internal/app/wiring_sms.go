package app

// 短信通道配置解析:biz_params(sms.* keys,后台短信配置页)优先,env 凭据兜底。
// 独立文件避免 wiring.go 膨胀;仅装配期使用。

import (
	"context"

	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/secretbox"
	"github.com/ymm-001/boss/internal/pkg/sms"
)

// smsConfigResolver 读取 biz_params → 明文 ChannelConfig;读库失败回退 env 形态。
func smsConfigResolver(svc user.Service, cfg *config.Config) func(context.Context) (sms.ChannelConfig, error) {
	return func(ctx context.Context) (sms.ChannelConfig, error) {
		out := sms.ChannelConfig{
			Enabled:         true,
			AccessKeyID:     cfg.SMS.AccessKeyID,
			AccessKeySecret: cfg.SMS.AccessKeySecret,
			From:            cfg.SMS.From,
		}
		list, err := svc.ListParams(ctx)
		if err != nil {
			return out, nil // DB 不可读:退回 env 兜底,不阻塞发送
		}
		stored := map[string]string{}
		for _, p := range list {
			stored[p.Key] = p.Value
		}
		smsApplyParams(&out, stored)
		return out, nil
	}
}

// smsApplyParams DB 值覆盖 env 兜底;secret 解密失败视为未配置。
func smsApplyParams(out *sms.ChannelConfig, stored map[string]string) {
	if v, ok := stored["sms.enabled"]; ok {
		out.Enabled = v != "false"
	}
	if v := stored["sms.accessKeyId"]; v != "" {
		out.AccessKeyID = v
	}
	if v := stored["sms.accessKeySecret"]; v != "" {
		if plain, err := secretbox.Open(v); err == nil {
			out.AccessKeySecret = plain
		}
	}
	if v := stored["sms.from"]; v != "" {
		out.From = v
	}
	out.TemplateCN = stored["sms.template.cn"]
	out.TemplateMY = stored["sms.template.my"]
}
