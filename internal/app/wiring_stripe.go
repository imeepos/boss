package app

// Stripe 卡收单通道配置解析:biz_params(stripe.* keys,后台支付配置页)优先,env 凭据兜底。
// 60s 热生效;未配置/未启用 → 发起端点 400、webhook 503,与既有"密钥未配即降级"裁定一致。

import (
	"context"
	"errors"
	"time"

	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/secretbox"
	"github.com/ymm-001/boss/internal/pkg/stripe"
)

// stripeConfigResolver 读取 biz_params → 明文 Config;读库失败回退 env 形态。
func stripeConfigResolver(lister paramLister, cfg *config.Config) func(context.Context) (stripe.Config, error) {
	return func(ctx context.Context) (stripe.Config, error) {
		out := stripe.Config{
			Enabled:       true,
			APIKey:        cfg.Stripe.APIKey,
			WebhookSecret: cfg.Stripe.WebhookSec,
			Currency:      cfg.Stripe.Currency,
			APIBaseURL:    cfg.Stripe.APIBaseURL,
		}
		list, err := lister.ListParams(ctx)
		if err != nil {
			return out, nil // DB 不可读:退回 env 兜底,不阻塞收单
		}
		stored := make(map[string]string, len(list))
		for _, p := range list {
			stored[p.Key] = p.Value
		}
		stripeApplyParams(&out, stored)
		return out, nil
	}
}

// stripeApplyParams DB 值覆盖 env 兜底;secret 解密失败视为未配置。
func stripeApplyParams(out *stripe.Config, stored map[string]string) {
	if v, ok := stored["stripe.enabled"]; ok {
		out.Enabled = v != "false"
	}
	if v := stored["stripe.apiKey"]; v != "" {
		if plain, err := secretbox.Open(v); err == nil {
			out.APIKey = plain
		}
	}
	if v := stored["stripe.publishableKey"]; v != "" {
		out.PublishableKey = v
	}
	if v := stored["stripe.webhookSecret"]; v != "" {
		if plain, err := secretbox.Open(v); err == nil {
			out.WebhookSecret = plain
		}
	}
	if v := stored["stripe.currency"]; v != "" {
		out.Currency = v
	}
	if v := stored["stripe.apiBaseUrl"]; v != "" {
		out.APIBaseURL = v
	}
}

// wireStripe 装配 Stripe 通道:动态配置驱动(DB/env 任一来源,60s 热生效)。
func wireStripe(app *Application, lister paramLister, cfg *config.Config) {
	dyn := stripe.NewDynamic(stripeConfigResolver(lister, cfg))
	app.Stripe = dyn
	app.PayGateway = billing.NewPaymentGatewayRegistry(stripeDynamicGateway{dyn: dyn})
	app.ReconSources.Register("stripe", stripeDynamicSource{dyn: dyn}) // 自动对账渠道源
}

// stripeDynamicGateway stripe.Dynamic → billing.PaymentGateway(每次调用按当前配置取客户端)。
type stripeDynamicGateway struct{ dyn *stripe.Dynamic }

func (g stripeDynamicGateway) Channel() string { return "stripe" }

func (g stripeDynamicGateway) CreateIntent(ctx context.Context, payNo string, cents int64, meta map[string]string) (billing.PayIntent, error) {
	c, err := g.dyn.Client(ctx)
	if err != nil {
		return billing.PayIntent{}, err
	}
	it, err := c.CreateIntent(ctx, payNo, cents, meta)
	if err != nil {
		return billing.PayIntent{}, err
	}
	return billing.PayIntent{IntentID: it.ID, ClientSecret: it.ClientSecret,
		AmountCents: it.AmountCents, Currency: it.Currency}, nil
}

func (g stripeDynamicGateway) CreateCheckout(ctx context.Context, payNo string, cents int64, meta map[string]string,
	successURL, cancelURL string) (billing.PayCheckout, error) {
	c, err := g.dyn.Client(ctx)
	if err != nil {
		return billing.PayCheckout{}, err
	}
	s, err := c.CreateCheckoutSession(ctx, payNo, cents, meta, successURL, cancelURL)
	if err != nil {
		return billing.PayCheckout{}, err
	}
	return billing.PayCheckout{SessionID: s.ID, URL: s.URL}, nil
}

// stripeDynamicSource stripe.Dynamic → billing.ChannelSource(对账拉单按当前配置;未配置跳过)。
type stripeDynamicSource struct{ dyn *stripe.Dynamic }

func (s stripeDynamicSource) PullStatement(ctx context.Context, _ string, day time.Time) ([]billing.ChannelStatementRow, error) {
	c, err := s.dyn.Client(ctx)
	if err != nil {
		if errors.Is(err, stripe.ErrDisabled) {
			return nil, nil // 通道未配置:不参与对账,不中断批次
		}
		return nil, err
	}
	rows, err := c.ListDayStatements(ctx, day)
	if err != nil {
		return nil, err
	}
	out := make([]billing.ChannelStatementRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, billing.ChannelStatementRow{ChannelRef: r.PayNo, Amount: r.Amount})
	}
	return out, nil
}
