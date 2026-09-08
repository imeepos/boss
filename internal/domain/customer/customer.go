package customer

import "time"

// Customer 客户档案(REQ-CRM-001 一人一档)。
// 字段权威:docs/contract/fields.md 2.1;企业多联系人/站点属扩展,届时加 contact 子表。
type Customer struct {
	ID              int64     `json:"id"`
	CustomerCode    string    `json:"customerCode"` // 用户码:展示冗余,前缀 C-,对账仍以 ID 为准(adopted 2026-08-21)
	Name            string    `json:"name"`
	Phone           string    `json:"phone"`
	IdType          string    `json:"idType"` // 身份证/护照/营业执照
	IdNo            string    `json:"idNo"`
	RealNameStatus  string    `json:"realNameStatus"` // VERIFIED / PENDING
	ServiceStatus   string    `json:"serviceStatus"`  // ACTIVE / ARREARS / SUSPENDED
	AddressID       int64     `json:"addressId"`
	LegalEntityID   int64     `json:"legalEntityId"`   // 归属运营主体(品牌=legal_entities)
	LegalEntityName string    `json:"legalEntityName"` // 归属运营主体名称(读取时 JOIN 现值,data-relations §6.1)
	RegionID        int64     `json:"regionId"`        // 地址所在经营区域快照(regions.id)
	RegionName      string    `json:"regionName"`      // 区域名快照:改名不改历史
	CreatedAt       time.Time `json:"createdAt"`
}

// CustomerQuery 列表过滤条件(管理后台 customer.html 列表)。
type CustomerQuery struct {
	NameKeyword   string
	Phone         string
	Status        string
	LegalEntityID int64
	RegionScope   string
	Limit         int
	Offset        int
}
