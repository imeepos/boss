package customer

import "time"

// Customer 客户档案(REQ-CRM-001 一人一档)。
// 字段权威:docs/contract/fields.md 2.1;企业多联系人/站点属扩展,届时加 contact 子表。
type Customer struct {
	ID             int64
	Name           string
	Phone          string
	IdType         string // 身份证/护照/营业执照
	IdNo           string
	RealNameStatus string // VERIFIED / PENDING
	ServiceStatus  string // ACTIVE / ARREARS / SUSPENDED
	AddressID      int64
	LegalEntityID  int64  // 归属运营主体(品牌=legal_entities)
	RegionPath     string // 区域 ltree 路径
	CreatedAt      time.Time
}

// CustomerQuery 列表过滤条件(管理后台 customer.html 列表)。
type CustomerQuery struct {
	NameKeyword string
	Phone       string
	Status      string
	Limit       int
	Offset      int
}
