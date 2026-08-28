package billing

import (
	"context"
	"time"
)

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

// 缴费流水 Payment 状态枚举(对齐 terms.md §4 payment.status)。
// 命名空间前缀 PaymentStatusXxx 避免与其他域(Coupon/Arrears...)冲突。
const (
	PaymentStatusSuccess  = "SUCCESS"  // 缴费/充值成功,落到余额或账单勾销
	PaymentStatusFailed   = "FAILED"   // 通道失败,仅留痕不动账,排查用
	PaymentStatusRefunded = "REFUNDED" // 已退款(财务侧冲账),不影响余额正向入账
)

// Payment 缴费/充值流水(充值无账单,bill_id 为 NULL → BillID=0)。
type Payment struct {
	ID         int64   `json:"id"`
	PayNo      string  `json:"payNo"`
	BillID     int64   `json:"billId"`
	CustomerID int64   `json:"customerId"` // 充值流水归属;账单流水可缺省(按账单回查)
	Amount     float64 `json:"amount"`
	Method     string  `json:"method"`   // wechat/alipay/card/cash
	Status     string  `json:"status"`   // SUCCESS/FAILED/REFUNDED
	CouponID   string  `json:"couponId"` // 可选:缴费抵扣券,核销与落账同事务
	// 柜面凭证要素(000167,纪要 2026-08-28):网点/柜台班次/操作员;
	// method 管资金通道,三列管人员归因(柜面现金 cash、扫码 wechat/alipay、POS card)。
	SiteName     string `json:"siteName"`
	CounterCode  string `json:"counterCode"`
	OperatorName string `json:"operatorName"` // 服务端取登录态,前端不传
	// 退款留痕(000112):全额退款后 reason/时间落流水;未退为空/nil。
	RefundReason string     `json:"refundReason"`
	RefundedAt   *time.Time `json:"refundedAt,omitempty"`
}

// DailyCashRow 柜台日结汇总行(按网点+操作员聚合当日 cash 流水;周敏口径:
// 收入/退款分列,退款按流水发生日归属——跨日冲销不得污染当日实点勾对)。
type DailyCashRow struct {
	SiteName      string   `json:"siteName"`
	OperatorName  string   `json:"operatorName"`
	InAmount      float64  `json:"inAmount"`                // 当日 cash SUCCESS 合计
	RefundAmount  float64  `json:"refundAmount"`            // 当日 cash REFUNDED 合计
	NetAmount     float64  `json:"netAmount"`               // 净额=收入-退款,仅汇总展示
	CountedAmount *float64 `json:"countedAmount,omitempty"` // 当日已回填钱箱实点
}

// DailyClosing 日结实点回填请求。
type DailyClosing struct {
	Date          string  `json:"date"` // YYYY-MM-DD
	SiteName      string  `json:"siteName"`
	OperatorName  string  `json:"operatorName"`
	CountedAmount float64 `json:"countedAmount"` // 钱箱实点
	CreatedBy     string  `json:"createdBy"`     // 回填人(服务端取登录态)
}

// DailyClosingResult 回填结果:差异=系统净额-实点;不平不阻塞回填,
// 差异经 [paycheck] DIFF 日志留痕供人工追缴/盘点。
type DailyClosingResult struct {
	ID           int64   `json:"id"`
	SystemAmount float64 `json:"systemAmount"` // 回填时系统净额快照
	DiffAmount   float64 `json:"diffAmount"`
	Balanced     bool    `json:"balanced"`
}

// PaymentReceipt 落账回执(含券抵扣明细)。
type PaymentReceipt struct {
	PaymentID     int64   `json:"paymentId"`
	Amount        float64 `json:"amount"`        // 实收(元)
	DeductedCents int64   `json:"deductedCents"` // 券抵扣(分),0=无券
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
	// PaymentExistsByPayNo 同 pay_no 流水是否已存在(渠道重投幂等直查)。
	PaymentExistsByPayNo(ctx context.Context, payNo string) (bool, error)
	// RecordTopup 充值落账:无账单流水 + 余额增加同事务,pay_no 唯一幂等;
	// 仅 SUCCESS 增余额,FAILED 只留痕不动余额。
	RecordTopup(ctx context.Context, p Payment) (int64, error)
	// RecordPayment 收款落账:缴费流水 + 账单置 PAID 同事务,pay_no 唯一幂等。
	RecordPayment(ctx context.Context, p Payment) (int64, error)
	// DailyCashSummary 柜台日结汇总:按网点+操作员聚合指定日期 cash 流水
	// (收入=SUCCESS 合计,退款=REFUNDED 合计,净额仅汇总),附当日已回填实点。
	DailyCashSummary(ctx context.Context, date string) ([]DailyCashRow, error)
	// SaveDailyClosing 柜台日结实点回填:UPSERT 当日快照并返回系统净额与差异;
	// 不平不阻塞回填,差异由 handler 输出 [paycheck] DIFF 可 grep 日志。
	SaveDailyClosing(ctx context.Context, cl DailyClosing) (DailyClosingResult, error)
	// CashPaymentsByDate 指定日期 cash 流水逐笔(日结报表下钻,含 REFUNDED 凭证)。
	CashPaymentsByDate(ctx context.Context, date string) ([]Payment, error)
	// RecordPaymentWithCoupon 带券缴费:CouponID 非空时同事务核销(promotion 注入),
	// payments.amount 记实收,抵扣额见回执 DeductedCents。
	RecordPaymentWithCoupon(ctx context.Context, p Payment) (PaymentReceipt, error)
	// RefundPayment 全额退款(000112):流水置 REFUNDED 留痕,账单无其他在流水时回 UNPAID;
	// 仅 SUCCESS 可退;发票不自动作废(人工 void/reissue)。
	RefundPayment(ctx context.Context, paymentID int64, reason string) (*Payment, error)
	// GenerateBills 出账:为在网客户按账期批量生成账单(金额=产品基础月费成交价快照),幂等。
	// 返回本次新生成账单数。区域调价覆盖(region_offers)待 lo_account 补齐 region_path 后接入。
	GenerateBills(ctx context.Context, period string) (int, error)
}
