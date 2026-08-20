package billing

import "context"

// Bill 账单(客户×账期唯一,金额=订单成交价快照)。
type Bill struct {
	BillID          int64   `json:"billId"`
	BillNo          string  `json:"billNo"`
	CustomerID      int64   `json:"customerId"`
	CustomerName    string  `json:"customerName"` // 快照
	LegalEntityID   int64   `json:"legalEntityId"`
	LegalEntityName string  `json:"legalEntityName"`
	RegionID        int64   `json:"regionId"`
	RegionName      string  `json:"regionName"`
	Period          string  `json:"period"` // 账期,如 2026-08
	Amount          float64 `json:"amount"`
	Status          string  `json:"status"` // UNPAID/PAID/OVERDUE
}

// Payment 缴费/充值流水(充值无账单,bill_id 为 NULL → BillID=0)。
type Payment struct {
	ID         int64   `json:"id"`
	PayNo      string  `json:"payNo"`
	BillID     int64   `json:"billId"`
	CustomerID int64   `json:"customerId"` // 充值流水归属;账单流水可缺省(按账单回查)
	Amount     float64 `json:"amount"`
	Method     string  `json:"method"` // wechat/alipay/card/cash
	Status     string  `json:"status"` // SUCCESS/FAILED/REFUNDED
}

// BillingService 计费账务域服务口(阶段5):出账/缴费。
type BillingService interface {
	ListBills(ctx context.Context, customerID int64) ([]Bill, error)
	CreateBill(ctx context.Context, b Bill) (int64, error)
	GetBill(ctx context.Context, id int64) (*Bill, error)
	ListPayments(ctx context.Context, billID int64) ([]Payment, error)
	// ListPaymentsByCustomer 按客户聚合缴费+充值流水(customer_id 优先,账单归属兜底)。
	ListPaymentsByCustomer(ctx context.Context, customerID int64) ([]Payment, error)
	CreatePayment(ctx context.Context, p Payment) (int64, error)
	// RecordPayment 收款落账:缴费流水 + 账单置 PAID 同事务,pay_no 唯一幂等。
	RecordPayment(ctx context.Context, p Payment) (int64, error)
	// GenerateBills 出账:为在网客户按账期批量生成账单(金额=产品基础月费成交价快照),幂等。
	// 返回本次新生成账单数。区域调价覆盖(region_offers)待 lo_account 补齐 region_path 后接入。
	GenerateBills(ctx context.Context, period string) (int, error)
}
