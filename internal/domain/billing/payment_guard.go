package billing

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrInvalidMethod 缴费方式白名单外哨兵:脏 method 直落库会毁渠道对账
// (纪要 2026-08-28-柜面现金收款,陈磊必补项)。
var ErrInvalidMethod = errors.New("billing: invalid payment method")

// paymentMethods 缴费方式白名单(terms.md §4)。柜面归类按资金通道:
// 现金 cash、扫码 wechat/alipay、POS 收单 card;offline 专属师傅个人代收,
// 柜面不得占用(周敏裁定,2026-08-28)。
var paymentMethods = map[string]bool{
	"wechat": true, "alipay": true, "card": true, "cash": true, "offline": true,
}

// validMethod 校验缴费方式白名单。
func validMethod(m string) bool { return paymentMethods[m] }

// genPayNo 兜底流水号:PAY-<yyyyMMddHHmmss>-<4位随机>;调用方未传 pay_no 时防空值撞号。
// 幂等语义仍以显式传单号为准(重复提交同号仅落一条,苏婉验收③)。
func genPayNo() string {
	var b [2]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("PAY-%s-%04X", time.Now().Format("20060102150405"), b)
}

// ParseCounterSites 解析网点主数据清单(biz_params counter.sites,JSON 字符串数组);
// 空串/非法 JSON 返回空切片——网点下拉数据源,主数据经业务参数页维护。
func ParseCounterSites(v string) []string {
	sites := []string{}
	if v == "" {
		return sites
	}
	_ = json.Unmarshal([]byte(v), &sites)
	return sites
}
