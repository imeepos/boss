package app

// Stripe 卡收单通道配置解析:仅读 biz_params(stripe.* keys,后台支付配置页)。
// 60s 热生效;未配置/未启用 → 发起端点 400、webhook 503,与既有"密钥未配即降级"裁定一致;
// 2026-09-05 裁定:env 兜底 BOSS_STRIPE_* 移除(App 端收款方式也以 Stripe 通道配置为准)。

import (
	"context"
	"errors"
	"time"

	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/stripe"
)

// stripeConfigResolver 读取 biz_params → 明文 Config;读库失败按未配置形态返回(不阻塞收单)。
func stripeConfigResolver(lister paramLister) func(context.Context) (stripe.Config, error) {
	return func(ctx context.Context) (stripe.Config, error) {
		out := stripe.Config{Enabled: true}
		list, err := lister.ListParams(ctx)
		if err != nil {
			return out, nil // DB 不可读:按未配置(无凭据)返回,不阻塞收单
		}
		stored := make(map[string]string, len(list))
		for _, p := range list {
			stored[p.Key] = p.Value
		}
		stripeApplyParams(&out, stored)
		return out, nil
	}
}

// stripeApplyParams DB 值应用;secret 解密失败视为未配置。
func stripeApplyParams(out *stripe.Config, stored map[string]string) {
	if v, ok := stored["stripe.enabled"]; ok {
		out.Enabled = v != "false"
	}
	if v := stored["stripe.apiKey"]; v != "" {
		out.APIKey = decryptConfigSecret("stripe.apiKey", v)
	}
	if v := stored["stripe.publishableKey"]; v != "" {
		out.PublishableKey = v
	}
	if v := stored["stripe.webhookSecret"]; v != "" {
		out.WebhookSecret = decryptConfigSecret("stripe.webhookSecret", v)
	}
	if v := stored["stripe.currency"]; v != "" {
		out.Currency = v
	}
	if v := stored["stripe.apiBaseUrl"]; v != "" {
		out.APIBaseURL = v
	}
	if v := stored["stripe.webhookUrl"]; v != "" {
		out.WebhookURL = v
	}
}

// StripeReady Stripe 卡收单通道是否可用(后台配置页已配密钥且启用;未配置则收款方式默认线下)。
// 2026-09-05 起凭据仅存 biz_params(env BOSS_STRIPE_* 兜底已移除),60s 热生效。
func (a *Application) StripeReady(ctx context.Context) bool {
	return a.Stripe != nil && a.Stripe.Configured(ctx)
}

// wireStripe 装配 Stripe 通道:动态配置驱动(DB,60s 热生效)。
func wireStripe(app *Application, lister paramLister) {
	dyn := stripe.NewDynamic(stripeConfigResolver(lister))
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
