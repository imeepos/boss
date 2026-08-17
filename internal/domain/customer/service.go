package customer

import (
	"context"
	"errors"
)

// ErrCustomerNotFound 客户不存在。
var ErrCustomerNotFound = errors.New("customer: not found")

// CustomerService 客户域服务口(阶段2)。
// 契约:CT-001 客户建档,成功后客户主数据对计费/订单/客服/门户可见(本切片仅发领域事件值,不接 Kafka)。
type CustomerService interface {
	// Create 建档,返回自增 id;成功后发 CustomerCreated 事件值。
	Create(ctx context.Context, c Customer) (int64, error)
	// Get 按 id 查客户。
	Get(ctx context.Context, id int64) (*Customer, error)
	// List 按条件分页查询。
	List(ctx context.Context, q CustomerQuery) ([]Customer, error)
}
