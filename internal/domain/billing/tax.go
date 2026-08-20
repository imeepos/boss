package billing

import (
	"context"
	"errors"
	"time"
)

// ARN 术语(TAX-004):对外单据连续编号,发票前缀 INV-、收据前缀 OR-。

// ErrDuplicateInvoice 账单已有在发票(CT-007 幂等:同账期不重复开票)。
var ErrDuplicateInvoice = errors.New("billing: invoice already issued for bill")

// ErrIllegalInvoiceTransition 非法发票状态迁移(如重复作废)。
var ErrIllegalInvoiceTransition = errors.New("billing: illegal invoice transition")

// Invoice 电子发票(普票)。系统内记录=开票意图+留痕;法定效力依赖税局回执(tax_no)。
type Invoice struct {
	ID           int64   `json:"id"`
	InvoiceNo    string  `json:"invoiceNo"` // 内部流水号(原 ARN),如 INV-00000001
	BillID       int64   `json:"billId"`
	BillNo       string  `json:"billNo"`
	CustomerID   int64   `json:"customerId"`
	CustomerName string  `json:"customerName"` // 快照
	Title        string  `json:"title"`        // 发票抬头,默认客户名
	NetAmount    float64 `json:"netAmount"`    // 不含税净额(=账单金额)
	VatRate      float64 `json:"vatRate"`      // VAT 12%
	VatAmount    float64 `json:"vatAmount"`    // = ROUND(net*rate,2)
	TotalAmount  float64 `json:"totalAmount"`  // = net + vat(TAX-002)
	Status       string  `json:"status"`       // ISSUED/VOIDED
	VoidReason   string  `json:"voidReason"`
	// 税局网关层(000049):与业务状态正交。
	TaxJurisdiction string     `json:"taxJurisdiction"` // CN/PH,空=未定
	TaxChannel      string     `json:"taxChannel"`      // manual/leqi/bir_eis
	TaxStatus       string     `json:"taxStatus"`       // PENDING/SUBMITTED/ISSUED/FAILED
	TaxNo           string     `json:"taxNo"`           // 税局票号,ISSUED 时非空
	TaxFailReason   string     `json:"taxFailReason"`
	IssuedAt        time.Time  `json:"issuedAt"`
	VoidedAt        *time.Time `json:"voidedAt,omitempty"`
}

// InvoiceRunResult 一个账期的自动开票结果。
type InvoiceRunResult struct {
	Issued    int     `json:"issued"`    // 本次新开票数
	FailedIDs []int64 `json:"failedIds"` // 开票失败的账单(人工处理清单)
}

// ErrInvoiceNotTaxable 税务状态不允许该操作(如已开具再提交)。
var ErrInvoiceNotTaxable = errors.New("billing: invoice tax status disallows operation")

// TaxService 发票税务域服务口(TAX/AG-04):自动开票 + 连续发号 + 作废重开 + 税局回执。
type TaxService interface {
	// ListInvoices 列出发票;customerID=0 返回全部。
	ListInvoices(ctx context.Context, customerID int64) ([]Invoice, error)
	// IssueInvoicesForPeriod 出账后自动开票:为该账期尚无在发票的账单逐张开票,
	// 幂等(已开票跳过);单张失败不中断批次,失败账单进 FailedIDs(重跑即重试)。
	IssueInvoicesForPeriod(ctx context.Context, period string) (InvoiceRunResult, error)
	// IssueInvoiceForBill 门户按单开票:按 customerID+billNo 定位账单并校验归属,
	// 未命中或不属于该客户返回 ErrNotFound。幂等:该账单已有非 VOIDED 发票时
	// 直接返回已有票(不占新号);否则走与批量开票同一内核新开一张。
	IssueInvoiceForBill(ctx context.Context, customerID int64, billNo string) (*Invoice, error)
	// VoidInvoice 作废发票:编号保留不回收,记录原因(TAX-003)。
	VoidInvoice(ctx context.Context, id int64, reason string) error
	// ReissueInvoice 重开:原票 VOID 保留编号 + 新票新号,返回新票。
	ReissueInvoice(ctx context.Context, id int64) (*Invoice, error)
	// GetInvoice 按主键查发票;未命中返回 ErrNotFound。
	GetInvoice(ctx context.Context, id int64) (*Invoice, error)
	// BackfillTaxNo 人工通道回填:运营者在税局平台开具后登记税局票号。
	// 仅 PENDING/FAILED/SUBMITTED 可回填;回填即税务状态 ISSUED。
	BackfillTaxNo(ctx context.Context, id int64, taxNo string) error
	// MarkTaxResult 落税局回执(网关同步返回/异步轮询后调用)。
	MarkTaxResult(ctx context.Context, id int64, r TaxReceipt) error
}
