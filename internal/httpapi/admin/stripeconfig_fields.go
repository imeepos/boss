package adminapi

// stripe-config 字段定义:Stripe 卡收单通道(apiKey/whsec)与选项(pk/currency/apiBase)。
// 与 auth/sms/realid 同一套 biz_params 存储/secretbox 加密/掩码回显约定。

// stripeFields 全部字段;groups: channel(通道)/webhook(回调)。
var stripeFields = []authField{
	// 通道:Stripe 卡收单
	{Key: "stripe.enabled", Group: "channel", Default: "true"},
	{Key: "stripe.apiKey", Group: "channel", Secret: true},     // sk_...
	{Key: "stripe.publishableKey", Group: "channel"},           // pk_...(前端 Stripe.js 预留)
	{Key: "stripe.currency", Group: "channel", Default: "php"}, // 记账币种小写
	{Key: "stripe.apiBaseUrl", Group: "channel"},               // 覆盖 API 地址(测试/代理),空=官方

	// 回调:webhook 验签
	{Key: "stripe.webhookSecret", Group: "webhook", Secret: true}, // whsec_...
}

// stripeFieldByGroup 组内字段(channel/webhook)。
func stripeFieldByGroup(group string) []authField {
	out := make([]authField, 0, len(stripeFields))
	for _, f := range stripeFields {
		if f.Group == group {
			out = append(out, f)
		}
	}
	return out
}

// stripeFieldByKey 全字段索引。
func stripeFieldByKey(key string) (authField, bool) {
	for _, f := range stripeFields {
		if f.Key == key {
			return f, true
		}
	}
	return authField{}, false
}

// stripeTestRequired channel 组自检必填(凭据在 channel、验签密钥在 webhook,横跨两组)。
var stripeTestRequired = []string{"stripe.apiKey", "stripe.webhookSecret"}
