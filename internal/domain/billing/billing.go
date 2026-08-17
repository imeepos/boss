package billing

import "context"

// Bill 账单(客户×账期唯一,金额=订单成交价快照)。
type Bill struct {
	BillID          int64
	BillNo          string
	CustomerID      int64
	CustomerName    string // 快照
	LegalEntityID   int64
	LegalEntityName string
	RegionID        int64
	RegionName      string
	Period          string // 账期,如 2026-08
	Amount          float64
	Status          string // UNPAID/PAID/OVERDUE
}

// Payment 缴费流水(针对账单的收款记录)。
type Payment struct {
	ID     int64
	PayNo  string
	BillID int64
	Amount float64
	Method string // wechat/alipay/card/cash
	Status string // SUCCESS/FAILED/REFUNDED
}

// BillingService 计费账务域服务口(阶段5):出账/缴费。
type BillingService interface {
	ListBills(ctx context.Context, customerID int64) ([]Bill, error)
	CreateBill(ctx context.Context, b Bill) (int64, error)
	GetBill(ctx context.Context, id int64) (*Bill, error)
	ListPayments(ctx context.Context, billID int64) ([]Payment, error)
	CreatePayment(ctx context.Context, p Payment) (int64, error)
}
