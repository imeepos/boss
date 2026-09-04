package monthly

import (
	"context"
	"errors"
	"regexp"
)

// 表标识(admin 路由段与 import task kind 共用)。
const (
	TableUserRevenue     = "user-revenue"
	TableNetworkDelivery = "network-delivery"
	TableFinanceCost     = "finance-cost"
)

// ErrInvalidInput 月份/区域/数值校验失败(42200)。
var ErrInvalidInput = errors.New("monthly: invalid input")

// ErrUnknownTable 表标识不存在(40400)。
var ErrUnknownTable = errors.New("monthly: unknown table")

// ErrNotFound 月度行不存在(40400)。
var ErrNotFound = errors.New("monthly: not found")

// monthRe YYYY-MM 权威格式(模板规范:如 2026-08)。
var monthRe = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])$`)

// Region 区域白名单行(51 个标准 Barangay,种子见迁移 000181)。
type Region struct {
	Region    string `json:"region"`
	Active    bool   `json:"active"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// UserRevenue 用户与收入事实(期末在用/主营总收入为派生列,服务端计算)。
type UserRevenue struct {
	Month             string `json:"month"`
	Region            string `json:"region"`
	OpeningActive     int64  `json:"openingActive"`
	NewUsers          int64  `json:"newUsers"`
	ChurnedUsers      int64  `json:"churnedUsers"`
	AdjustedUsers     int64  `json:"adjustedUsers"`
	ClosingActive     int64  `json:"closingActive"` // 派生=期初+新增-离网+调整
	BroadbandRevenue  int64  `json:"broadbandRevenue"`
	ValueAddedRevenue int64  `json:"valueAddedRevenue"`
	OnetimeCharge     int64  `json:"onetimeCharge"`
	DiscountAmount    int64  `json:"discountAmount"`
	RefundReversal    int64  `json:"refundReversal"`
	TotalRevenue      int64  `json:"totalRevenue"` // 派生=宽带+增值+一次性-优惠-退款
	UpdatedAt         string `json:"updatedAt,omitempty"`
}

// NetworkDelivery 网络与交付事实。
type NetworkDelivery struct {
	Month             string `json:"month"`
	Region            string `json:"region"`
	InstallRequests   int64  `json:"installRequests"`
	OntimeCompletions int64  `json:"ontimeCompletions"`
	PortsDeployed     int64  `json:"portsDeployed"`
	PortsActive       int64  `json:"portsActive"`
	FaultReports      int64  `json:"faultReports"`
	RepairHours       int64  `json:"repairHours"`
	UpdatedAt         string `json:"updatedAt,omitempty"`
}

// FinanceCost 财务与成本事实。
type FinanceCost struct {
	Month            string `json:"month"`
	Region           string `json:"region"`
	InvoicedAmount   int64  `json:"invoicedAmount"`
	CollectedAmount  int64  `json:"collectedAmount"`
	ReceivableEnding int64  `json:"receivableEnding"`
	DirectCost       int64  `json:"directCost"`
	FixedCost        int64  `json:"fixedCost"`
	CapexInvest      int64  `json:"capexInvest"`
	UpdatedAt        string `json:"updatedAt,omitempty"`
}

// Filter 列表筛选(month/region 可空=全部)+分页。
type Filter struct {
	Month    string
	Region   string
	Page     int
	PageSize int
}

// PageResult 分页信封(aaa AdminPageResult 同形)。
type PageResult[T any] struct {
	Items    []T `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

// RowError 导入行级错误(行号=文件内 1 基行,含表头行)。
type RowError struct {
	Line   int    `json:"line"`
	Reason string `json:"reason"`
}

// ImportResult 导入结果(表标识由调用方上下文携带)。
type ImportResult struct {
	Total    int        `json:"total"`
	Imported int        `json:"imported"`
	Failed   int        `json:"failed"`
	Errors   []RowError `json:"errors,omitempty"`
}

// Summary 月度汇总 KPI;分母为 0 时该率返回 nil(JSON null)不报错。
type Summary struct {
	Month              string   `json:"month"`
	ClosingActiveTotal int64    `json:"closingActiveTotal"`
	TotalRevenueTotal  int64    `json:"totalRevenueTotal"`
	OntimeRate         *float64 `json:"ontimeRate"`      // 及时完工÷装机申请
	PortUtilization    *float64 `json:"portUtilization"` // 在用÷部署
	CollectionRate     *float64 `json:"collectionRate"`  // 实际回款÷开票
	Arpu               *float64 `json:"arpu"`            // 总收入÷期末在用
}

// Service 月度填报域口(admin 使用;导入导出同契约)。
type Service interface {
	ListRegions(ctx context.Context) ([]Region, error)
	ListUserRevenue(ctx context.Context, f Filter) (PageResult[UserRevenue], error)
	UpsertUserRevenue(ctx context.Context, r UserRevenue) error
	ListNetworkDelivery(ctx context.Context, f Filter) (PageResult[NetworkDelivery], error)
	UpsertNetworkDelivery(ctx context.Context, r NetworkDelivery) error
	ListFinanceCost(ctx context.Context, f Filter) (PageResult[FinanceCost], error)
	UpsertFinanceCost(ctx context.Context, r FinanceCost) error
	ImportCSV(ctx context.Context, table string, data []byte) (ImportResult, error)
	ExportCSV(ctx context.Context, table, month string) ([]byte, error)
	Summary(ctx context.Context, month string) (Summary, error)
}

// ValidateMonth 权威格式 YYYY-MM。
func ValidateMonth(m string) error {
	if !monthRe.MatchString(m) {
		return ErrInvalidInput
	}
	return nil
}
