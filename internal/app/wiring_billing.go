package app

// prepaidCollector 包装 billing/portal 域为 order.PrepaidCollector 接口(环节4 当场收款)。
// 派 PAY 单号走 portal_seq(与门户充值同源),落缴费流水走 billing.CreatePayment。

import (
	"context"

	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/portal"
)

type prepaidCollector struct {
	bill   *billing.PGStore
	portal portal.Service
}

// Collect 预付费收款:PAY 单号 + 缴费流水(bill_id NULL + customer_id 归属)。
// method=cash:环节4 为人工确认收款口径,移动支付渠道接入后经 Amended 扩展。
func (c prepaidCollector) Collect(ctx context.Context, customerID int64, amount float64) error {
	payNo, err := c.portal.NextNo(ctx, "PAY")
	if err != nil {
		return err
	}
	_, err = c.bill.CreatePayment(ctx, billing.Payment{
		PayNo: payNo, CustomerID: customerID, Amount: amount,
		Method: "cash", Status: "SUCCESS",
	})
	return err
}
