package stripe

// Config 通道配置(resolve 返回的明文形态)。
// APIKey/WebhookSecret 为 Stripe 测试或生产密钥;biz_params 存储密文,resolve 已解密。
type Config struct {
	Enabled        bool
	APIKey         string // sk_...(发起意图/确认/对账)
	PublishableKey string // pk_...(前端 Stripe.js 预留,托管收银台不需要)
	WebhookSecret  string // whsec_...(回调验签)
	Currency       string // 记账币种小写,如 php
	APIBaseURL     string // 覆盖官方 API 地址(测试/代理),空=官方
}
