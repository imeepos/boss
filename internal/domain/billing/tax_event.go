package billing

import (
	"context"
	"time"
)

// 发票税局轨迹事件(与 invoice.status/tax_status 正交:记录"怎么到的当前态")。
const (
	TaxEventReceipt  = "RECEIPT"  // 网关回执(tax-submit 同步返回:ISSUED/FAILED/SUBMITTED)
	TaxEventBackfill = "BACKFILL" // 人工通道回填票号
	TaxEventVoid     = "VOID"     // 发票作废(发票维度,tax_status_after=当时税局状态)
	TaxEventReissue  = "REISSUE"  // 原票作废重开(事件挂原票)
)

// TaxEvent 发票税局轨迹行。OperatorAccountID=0 表示网关/系统自动动作。
type TaxEvent struct {
	ID                int64     `json:"id"`
	InvoiceID         int64     `json:"invoiceId"`
	Event             string    `json:"event"`
	TaxStatusAfter    string    `json:"taxStatusAfter"`
	TaxNo             string    `json:"taxNo"`
	FailReason        string    `json:"failReason"`
	ExternalID        string    `json:"externalId"`
	OperatorAccountID int64     `json:"operatorAccountId"`
	CreatedAt         time.Time `json:"createdAt"`
}

// TaxEventService 发票税局轨迹域服务口(追加 + 按发票回放)。
type TaxEventService interface {
	AppendTaxEvent(ctx context.Context, e TaxEvent) (int64, error)
	ListTaxEvents(ctx context.Context, invoiceID int64) ([]TaxEvent, error)
}
