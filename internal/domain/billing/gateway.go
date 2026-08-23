package billing

import "context"

// 税务属地与通道(TaxGateway)。系统内发票无独立法定效力,须经理税局通道开具,
// 回填税局票号(tax_no)后方为有效票据。多属地并存:CN 数电票 / PH BIR eIS。

// 税务属地。
const (
	TaxJurisdictionCN = "CN" // 中国大陆:数电票(电子发票服务平台/乐企)
	TaxJurisdictionPH = "PH" // 菲律宾:BIR eIS
)

// 开票通道。manual=人工在税局平台开具后回填;其余为预留适配器(资质/密钥就绪后落地)。
const (
	TaxChannelManual = "manual"
	TaxChannelLeqi   = "leqi"    // CN 乐企接口
	TaxChannelBIREIS = "bir_eis" // PH BIR eIS 报送
)

// 税务状态(与发票业务状态 status=ISSUED/VOIDED 正交:业务已出具但税务可仍待开具)。
const (
	TaxStatusPENDING   = "PENDING"   // 待开具(初始)
	TaxStatusSUBMITTED = "SUBMITTED" // 已提交税局,待回执
	TaxStatusIssued    = "ISSUED"    // 税局已开具(tax_no 回填)
	TaxStatusFailed    = "FAILED"    // 开具失败(tax_fail_reason 留痕,可重试)
	TaxStatusBlocked   = "BLOCKED"   // 外部资质/凭据不可用,不得伪造成功
)

// TaxReceipt 税局回执。
type TaxReceipt struct {
	TaxNo      string // 税局票号:CN 数电票 20 位 / PH BIR 回执号
	Status     string // TaxStatusIssued / TaxStatusFailed / TaxStatusSUBMITTED / TaxStatusBlocked
	FailReason string
	ExternalID string `json:"externalId"` // 外部回执/请求标识,用于重复与乱序幂等
}

// TaxGateway 税局开票网关:按属地各一个实现,实现方负责签名/报送/重试语义。
// manual 通道无网关(人工回填),实现不注册。
type TaxGateway interface {
	Jurisdiction() string                                       // CN / PH
	Channel() string                                            // leqi / bir_eis
	Issue(ctx context.Context, inv Invoice) (TaxReceipt, error) // 同步回执;异步回执由实现方轮询后回填
}

// TaxGatewayRegistry 属地→网关注册表;装配期注入,运行期只读。
type TaxGatewayRegistry struct {
	byJurisdiction map[string]TaxGateway
}

// NewTaxGatewayRegistry 注册零到多个网关;无网关=全人工模式(现状,不破坏)。
func NewTaxGatewayRegistry(gateways ...TaxGateway) *TaxGatewayRegistry {
	r := &TaxGatewayRegistry{byJurisdiction: make(map[string]TaxGateway, len(gateways))}
	for _, g := range gateways {
		r.byJurisdiction[g.Jurisdiction()] = g
	}
	return r
}

// Get 按属地取网关;未注册返回 nil(调用方走人工回填)。
func (r *TaxGatewayRegistry) Get(jurisdiction string) TaxGateway {
	if r == nil {
		return nil
	}
	return r.byJurisdiction[jurisdiction]
}
