package odn

import (
	"context"
	"errors"
	"math"
)

// 工程应付台账状态(P-INFRA-1 W6,迁移 000219;审查 F8;枚举登记 terms.md §4)。
const (
	APOpen    = "OPEN"    // 未付:净应付未收任何付款
	APPartial = "PARTIAL" // 部分付款:0 < 已付 < 净应付(分期同口径逐笔登记)
	APPaid    = "PAID"    // 已付清:已付 >= 净应付
	APVoided  = "VOIDED"  // 已冲销:来源结算单 VOIDED 同事务冲销,流水保留历史
)

// 付款方式(construction_payable_payments.method;登记口径,不动 billing 缴费 method 枚举)。
const (
	PayTransfer = "TRANSFER" // 银行转账
	PayCash     = "CASH"     // 现金
	PayCheque   = "CHEQUE"   // 支票
	PayOther    = "OTHER"    // 其他
)

var (
	// ErrPayableState 非法应付状态操作(VOIDED 再动/超余额付款/超应付核减等):40900。
	ErrPayableState = errors.New("odn: invalid payable state")
)

// Payable 工程应付(净应付/余额只读派生:净应付=应付-核减合计,余额=净应付-已付)。
type Payable struct {
	ID             int64   `json:"id"`
	PayableNo      string  `json:"payableNo"`
	SettlementID   int64   `json:"settlementId"`
	SettlementNo   string  `json:"settlementNo"`
	ProjectID      int64   `json:"projectId"`
	ProjectNo      string  `json:"projectNo"`
	ContractorID   int64   `json:"contractorId"`
	ContractorName string  `json:"contractorName"`
	PayableAmount  float64 `json:"payableAmount"`
	DeductedAmount float64 `json:"deductedAmount"` // 核减合计(派生)
	PaidAmount     float64 `json:"paidAmount"`     // 已付合计(派生)
	Balance        float64 `json:"balance"`        // 未付余额=净应付-已付(派生,VOIDED 可为负=超付)
	Status         string  `json:"status"`
	VoidReason     string  `json:"voidReason"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

// PayablePayment 付款流水登记行(部分付款与分期=多次登记至余额耗尽)。
type PayablePayment struct {
	ID        int64   `json:"id"`
	PayableID int64   `json:"payableId"`
	PaymentNo string  `json:"paymentNo"`
	Amount    float64 `json:"amount"`
	Method    string  `json:"method"`
	PaidAt    string  `json:"paidAt"`
	Reference string  `json:"reference"`
	Note      string  `json:"note"`
	CreatedBy int64   `json:"createdBy"`
	CreatedAt string  `json:"createdAt"`
}

// PayableDeduction 核减明细(append-only,原因必填,留痕可回放)。
type PayableDeduction struct {
	ID        int64   `json:"id"`
	PayableID int64   `json:"payableId"`
	Amount    float64 `json:"amount"`
	Reason    string  `json:"reason"`
	CreatedBy int64   `json:"createdBy"`
	CreatedAt string  `json:"createdAt"`
}

// PayableInvoice 发票登记行(纯登记,同应付内发票号唯一,不联动税局)。
type PayableInvoice struct {
	ID         int64   `json:"id"`
	PayableID  int64   `json:"payableId"`
	InvoiceNo  string  `json:"invoiceNo"`
	Amount     float64 `json:"amount"`
	InvoicedAt string  `json:"invoicedAt,omitempty"`
	Note       string  `json:"note"`
	CreatedBy  int64   `json:"createdBy"`
	CreatedAt  string  `json:"createdAt"`
}

// PayableDetail 应付详情聚合(单头+三类流水,单据链可回放)。
type PayableDetail struct {
	Payable    Payable            `json:"payable"`
	Payments   []PayablePayment   `json:"payments"`
	Deductions []PayableDeduction `json:"deductions"`
	Invoices   []PayableInvoice   `json:"invoices"`
}

// round2 金额按分位四舍五入(NUMERIC(14,2) 同口径),避免浮点漂移。
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// ComputePayableStatus 按应付/核减/已付派生台账状态(VOIDED 由冲销路径直写,不经此派生)。
func ComputePayableStatus(payable, deducted, paid float64) string {
	net := round2(payable - deducted)
	paid = round2(paid)
	switch {
	case paid <= 0:
		return APOpen
	case paid >= net:
		return APPaid
	default:
		return APPartial
	}
}

// ValidatePayRegister 付款登记纯校验:金额>0,方式合法,不超未付余额(分期逐笔登记口径)。
func ValidatePayRegister(method string, amount, balance float64) error {
	if amount <= 0 {
		return ErrInvalidInput
	}
	switch method {
	case PayTransfer, PayCash, PayCheque, PayOther:
	default:
		return ErrInvalidInput
	}
	if round2(amount) > round2(balance) {
		return ErrPayableState
	}
	return nil
}

// ValidateDeduct 核减纯校验:原因必填,金额>0,核减不超应付,核减后净应付不得低于已付(防超付)。
func ValidateDeduct(reason string, amount, payable, deducted, paid float64) error {
	if reason == "" || amount <= 0 {
		return ErrInvalidInput
	}
	if round2(deducted+amount) > round2(payable) {
		return ErrPayableState
	}
	if round2(paid) > round2(payable-deducted-amount) {
		return ErrPayableState
	}
	return nil
}

// PayableStore 工程应付存储口(PGStore 实现;接口收敛在 ODNService)。
type PayableStore interface {
	ListPayables(ctx context.Context, status string, projectID int64, limit int) ([]Payable, error)
	GetPayable(ctx context.Context, id int64) (*PayableDetail, error)
	RegisterPayablePayment(ctx context.Context, payableID, accountID int64, amount float64,
		method, paidAt, reference, note string) (*PayablePayment, error)
	DeductPayable(ctx context.Context, payableID, accountID int64, amount float64, reason string) (*PayableDeduction, error)
	RegisterPayableInvoice(ctx context.Context, payableID, accountID int64, amount float64,
		invoiceNo, invoicedAt, note string) (*PayableInvoice, error)
}
