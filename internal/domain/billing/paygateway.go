package billing

import "context"

// 支付渠道网关(裁定 2026-08-20 §4:支付 create/confirm/callback 接口收敛在域内,
// 实现为薄 HTTP adapter,系统侧自持状态:渠道只收单,记账事实源=payments+账单状态机)。

// PayIntent 渠道支付意图:发起支付时渠道返回,clientSecret 交前端渠道 SDK 拉起收银台。
type PayIntent struct {
	IntentID     string
	ClientSecret string
	AmountCents  int64
	Currency     string
}

// PaymentGateway 支付渠道网关:Channel 标识对齐对账批次 channel(如 stripe)。
// 回调侧不做网关方法:webhook 验签属渠道实现细节,由 httpapi 层调适配器完成。
type PaymentGateway interface {
	Channel() string
	// CreateIntent 发起收款;payNo 同时作为渠道幂等键与 metadata 回传字段。
	CreateIntent(ctx context.Context, payNo string, amountCents int64, metadata map[string]string) (PayIntent, error)
}

// PaymentGatewayRegistry 渠道→网关注册表;装配期注入,运行期只读。
type PaymentGatewayRegistry struct {
	byChannel map[string]PaymentGateway
}

// NewPaymentGatewayRegistry 注册零到多个网关;无网关=全部走既有模拟通道(不破坏)。
func NewPaymentGatewayRegistry(gateways ...PaymentGateway) *PaymentGatewayRegistry {
	r := &PaymentGatewayRegistry{byChannel: make(map[string]PaymentGateway, len(gateways))}
	for _, g := range gateways {
		r.byChannel[g.Channel()] = g
	}
	return r
}

// Get 按渠道取网关;未注册返回 nil(调用方回退模拟直落账)。
func (r *PaymentGatewayRegistry) Get(channel string) PaymentGateway {
	if r == nil {
		return nil
	}
	return r.byChannel[channel]
}
