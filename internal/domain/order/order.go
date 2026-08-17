package order

import "time"

// Order 订单主表(REQ-ORD-002:12 环节可跟踪)。
// 字段权威:docs/contract/fields.md 3.1;状态枚举见 docs/contract/terms.md 第 3 节。
type Order struct {
	ID            int64
	OrderNo       string // ORD-20250817-001
	CustomerID    int64
	OfferID       int64 // 产品 → product_offers(fields.md §3.1)
	AddressID     int64
	Stage         int8   // 1~12(见 terms.md 第 1 节)
	Status        string // PENDING / RESERVED / INSTALLING / DONE
	ChannelID     int64  // 渠道,REQ-ORD-006 必填不可改(CH 域待建,暂以占位 ID)
	LegalEntityID int64  // 品牌=运营主体(legal_entities, 品牌隔离最小单元)
	RegionPath    string
	CreatedAt     time.Time
}

// StageLog 环节时间轴(order.html「环节时间轴」表)。
type StageLog struct {
	ID         int64
	OrderID    int64
	Stage      int8
	Result     string // DONE / DOING / PENDING
	Retries    int
	FinishedAt *time.Time
}

// SubmitReq 下单请求(环节1)。
type SubmitReq struct {
	CustomerID    int64
	OfferID       int64
	AddressID     int64
	ChannelID     int64 // 必填,不可改(CH 域待建,暂以占位 ID)
	LegalEntityID int64 // 品牌=运营主体(legal_entities)
	RegionPath    string
}
