package app

// customerLookup 把 customer.CustomerService.Get 适配为 order.CustomerLookup.Exists。
// (自 wiring.go 平移,零行为变更;该文件已至行数红线。)

import (
	"context"
	"errors"

	"github.com/ymm-001/boss/internal/domain/customer"
)

type customerLookup struct{ svc customer.CustomerService }

func (c customerLookup) Exists(ctx context.Context, id int64) (bool, error) {
	_, err := c.svc.Get(ctx, id)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, customer.ErrCustomerNotFound) {
		return false, nil
	}
	return false, err
}
