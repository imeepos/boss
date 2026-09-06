package odn

import (
	"context"
	"errors"
)

// 绑定领域错误(P3,T13,迁移 000201)。
var (
	// ErrPortNotInService 绑定要求端口处于 IN_SERVICE(未开通不可绑定)。
	ErrPortNotInService = errors.New("odn: port not in service")
	// ErrBindingNotFound 绑定记录不存在。
	ErrBindingNotFound = errors.New("odn: binding not found")
)

// ODNBinding 逻辑-物理绑定事实(订单/逻辑口 ↔ ODN 物理口;售后与影响面反查的数据源)。
type ODNBinding struct {
	ID             int64  `json:"id"`
	PortID         int64  `json:"portId"`
	OrderID        int64  `json:"orderId"`
	ResourcePortID int64  `json:"resourcePortId"`
	Note           string `json:"note"`
	BoundBy        int64  `json:"boundBy"`
	BoundAt        string `json:"boundAt"`
}

// ValidateBinding 绑定参数:订单与端口必填。
func ValidateBinding(orderID, portID int64) error {
	if orderID <= 0 || portID <= 0 {
		return ErrInvalidCoverage
	}
	return nil
}

// BindingStore 绑定存储口(PGStore 实现)。
type BindingStore interface {
	BindPort(ctx context.Context, b ODNBinding) error
	UnbindPort(ctx context.Context, portID int64) error
	ListBindingsByPort(ctx context.Context, portID int64) ([]ODNBinding, error)
	ListBindingsByOrder(ctx context.Context, orderID int64) ([]ODNBinding, error)
}
