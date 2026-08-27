package app

// 推送通道配置解析:biz_params(push.* keys,后台推送配置页)优先,env 凭据兜底。
// 独立文件避免 wiring.go 膨胀;仅装配期使用。

import (
	"context"
	"strconv"

	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/push"
)

// pushConfigResolver 读取 biz_params → 明文 ChannelConfig;读库失败回退 env 形态。
func pushConfigResolver(svc user.Service, cfg *config.Config) func(context.Context) (push.ChannelConfig, error) {
	return func(ctx context.Context) (push.ChannelConfig, error) {
		out := push.ChannelConfig{
			Enabled:        true,
			Provider:       "jpush",
			ApnsProduction: true,
			APIURL:         push.DefaultAPIURL,
			AppKey:         cfg.JPush.AppKey,
			MasterSecret:   cfg.JPush.MasterSecret,
		}
		list, err := svc.ListParams(ctx)
		if err != nil {
			return out, nil // DB 不可读:退回 env 兜底,不阻塞装配
		}
		stored := map[string]string{}
		for _, p := range list {
			stored[p.Key] = p.Value
		}
		pushApplyParams(&out, stored)
		return out, nil
	}
}

// pushApplyParams DB 值覆盖 env 兜底;secret 解密失败视为未配置。
func pushApplyParams(out *push.ChannelConfig, stored map[string]string) {
	if v, ok := stored["push.enabled"]; ok {
		out.Enabled = v != "false"
	}
	if v := stored["push.provider"]; v != "" {
		out.Provider = v
	}
	if v := stored["push.jpush.appKey"]; v != "" {
		out.AppKey = v
	}
	if v := stored["push.jpush.masterSecret"]; v != "" {
		out.MasterSecret = decryptConfigSecret("push.jpush.masterSecret", v)
	}
	if v := stored["push.jpush.apiUrl"]; v != "" {
		out.APIURL = v
	}
	if v := stored["push.jpush.apnsProduction"]; v != "" {
		out.ApnsProduction = v != "false"
	}
	if v := stored["push.jpush.liveTime"]; v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			out.LiveTimeSec = n
		}
	}
}
