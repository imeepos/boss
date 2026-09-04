package monthly

// 派生列口径(模板灰色列「公式-勿填」的服务端实现,权威输入 docs/books 模板说明):
// 期末在用 = 期初在用 + 当月新增 - 当月离网 + 数据调整;
// 主营总收入 = 宽带收入 + 增值收入 + 一次性收费 - 优惠减免 - 退款冲销。
// 迁移 000181 以 GENERATED ALWAYS 存储列落地同口径,本文件供服务端组装响应/校验。

// ClosingActive 期末在用(派生)。
func ClosingActive(opening, added, churned, adjusted int64) int64 {
	return opening + added - churned + adjusted
}

// TotalRevenue 主营总收入(派生)。
func TotalRevenue(broadband, valueAdded, onetime, discount, refund int64) int64 {
	return broadband + valueAdded + onetime - discount - refund
}

// Ratio 比率(分母 0 返回 nil,JSON null;Summary 口径要求)。
func Ratio(num, den int64) *float64 {
	if den == 0 {
		return nil
	}
	v := float64(num) / float64(den)
	return &v
}
