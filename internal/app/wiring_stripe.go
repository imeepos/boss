package app

// Stripe 支付通道装配:密钥齐备才注册网关;薄适配器形态,域接口与渠道实现解耦。

import (
	"context"

	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/stripe"
)

// wireStripe 装配 Stripe 通道:密钥齐备才注册网关与验签上下文,否则维持模拟直落账。
func wireStripe(app *Application, cfg *config.Config) {
	app.StripeWebhook = stripe.Webhook{Secret: cfg.Stripe.WebhookSec}
	c := stripe.New(cfg.Stripe.APIKey, cfg.Stripe.Currency)
	if c == nil {
		app.PayGateway = billing.NewPaymentGatewayRegistry()
		return
	}
	if cfg.Stripe.APIBaseURL != "" {
		c.BaseURL = cfg.Stripe.APIBaseURL
	}
	app.PayGateway = billing.NewPaymentGatewayRegistry(stripeGateway{c: c})
}

// stripeGateway stripe.Client → billing.PaymentGateway 适配(结构不同型,显式转换)。
type stripeGateway struct{ c *stripe.Client }

func (g stripeGateway) Channel() string { return g.c.Channel() }

func (g stripeGateway) CreateIntent(ctx context.Context, payNo string, cents int64, meta map[string]string) (billing.PayIntent, error) {
	it, err := g.c.CreateIntent(ctx, payNo, cents, meta)
	if err != nil {
		return billing.PayIntent{}, err
	}
	return billing.PayIntent{IntentID: it.ID, ClientSecret: it.ClientSecret,
		AmountCents: it.AmountCents, Currency: it.Currency}, nil
}
