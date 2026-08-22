package order

import "time"

// Order 订单主表(REQ-ORD-002:12 环节可跟踪)。
// 字段权威:docs/contract/fields.md 3.1;状态枚举见 docs/contract/terms.md 第 3 节。
type Order struct {
	ID            int64     `json:"id"`
	OrderNo       string    `json:"orderNo"` // ORD-20250817-001
	CustomerID    int64     `json:"customerId"`
	OfferID       int64     `json:"offerId"` // 产品 → product_offers(fields.md §3.1)
	AddressID     int64     `json:"addressId"`
	Stage         int8      `json:"stage"`         // 1~12(见 terms.md 第 1 节)
	Status        string    `json:"status"`        // PENDING / RESERVED / INSTALLING / DONE
	ChannelID     int64     `json:"channelId"`     // 渠道 → channels,REQ-ORD-006 必填不可改
	LegalEntityID int64     `json:"legalEntityId"` // 品牌=运营主体(legal_entities, 品牌隔离最小单元)
	RegionPath    string    `json:"regionPath"`
	BillingMode   string    `json:"billingMode"` // PREPAID/POSTPAID(000102);空回退 POSTPAID
	BuyMonths     int       `json:"buyMonths"`   // 预缴月数(000104);0=按月缴
	GiftMonths    int       `json:"giftMonths"`  // 赠送月数(000104);环节4 收款按阶梯命中回填
	CreatedAt     time.Time `json:"createdAt"`
}

// 付费模式枚举(与 aaa 域对齐,terms.md §4);下单客户选定,环节6 继承到 lo_accounts。
const (
	BillingModePrepaid  = "PREPAID"
	BillingModePostpaid = "POSTPAID"
)

// StageLog 环节时间轴(order.html「环节时间轴」表)。
type StageLog struct {
	ID         int64      `json:"id"`
	OrderID    int64      `json:"orderId"`
	Stage      int8       `json:"stage"`
	Result     string     `json:"result"` // DONE / DOING / PENDING
	Retries    int        `json:"retries"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

// SubmitReq 下单请求(环节1)。
// LegalEntityID/RegionPath 为可选校验值:归属由安装地址服务端推导
// (adopted note 2026-08-20-order-legal-entity-by-address),传入值与推导冲突时拒单。
type SubmitReq struct {
	CustomerID    int64  `json:"customerId"`
	OfferID       int64  `json:"offerId"`
	AddressID     int64  `json:"addressId"`
	ChannelID     int64  `json:"channelId"`     // 必填,不可改(→ channels)
	LegalEntityID int64  `json:"legalEntityId"` // 可选校验:≠0 时须与地址推导一致
	RegionPath    string `json:"regionPath"`
	BillingMode   string `json:"billingMode"` // PREPAID/POSTPAID,空回退 POSTPAID;环节4 预付费当场收款
	BuyMonths     int    `json:"buyMonths"`   // 预缴月数,0=按月缴(环节4 收 1 个月);1~60
}
