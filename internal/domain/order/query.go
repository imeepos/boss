package order

import "time"

// OrderQuery 订单列表查询条件(承接 order.html 订单管理列表)。
type OrderQuery struct {
	Keyword    string // 订单号模糊
	Status     string // PENDING/RESERVED/INSTALLING/DONE;空=全部
	CustomerID int64  // 客户过滤;0=全部
	Limit      int    // <=0 视为不限
	Offset     int
}

// OrderListItem 订单列表读模型(联表展示客户/产品/地址名,字段与 order.html 列表列对齐)。
type OrderListItem struct {
	OrderNo   string    `json:"orderNo"`
	Customer  string    `json:"customer"`
	Product   string    `json:"product"`
	Address   string    `json:"address"`
	AddressID int64     `json:"addressId"`
	Stage     int8      `json:"stage"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}
