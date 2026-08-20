package app

// 实名核验通道配置解析:biz_params(realid.* keys,后台实名核验配置页)优先,env 凭据兜底。
// 未配置/未启用 → Dynamic 返回 ErrDisabled,提交落 PENDING 人工核验(裁定:自动不取代人工)。

import (
	"context"

	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/realid"
	"github.com/ymm-001/boss/internal/pkg/secretbox"
)

// realidConfigResolver 读取 biz_params → 明文 ChannelConfig;读库失败回退 env 形态。
func realidConfigResolver(svc user.Service, cfg *config.Config) func(context.Context) (realid.ChannelConfig, error) {
	return func(ctx context.Context) (realid.ChannelConfig, error) {
		out := realid.ChannelConfig{
			Enabled:         true,
			AccessKeyID:     cfg.RealID.AccessKeyID,
			AccessKeySecret: cfg.RealID.AccessKeySecret,
		}
		list, err := svc.ListParams(ctx)
		if err != nil {
			return out, nil // DB 不可读:退回 env 兜底,不阻塞核验
		}
		stored := map[string]string{}
		for _, p := range list {
			stored[p.Key] = p.Value
		}
		realidApplyParams(&out, stored)
		return out, nil
	}
}

// realidApplyParams DB 值覆盖 env 兜底;secret 解密失败视为未配置。
func realidApplyParams(out *realid.ChannelConfig, stored map[string]string) {
	if v, ok := stored["realid.enabled"]; ok {
		out.Enabled = v != "false"
	}
	if v := stored["realid.accessKeyId"]; v != "" {
		out.AccessKeyID = v
	}
	if v := stored["realid.accessKeySecret"]; v != "" {
		if plain, err := secretbox.Open(v); err == nil {
			out.AccessKeySecret = plain
		}
	}
	if v := stored["realid.endpoint"]; v != "" {
		out.Endpoint = v
	}
}
