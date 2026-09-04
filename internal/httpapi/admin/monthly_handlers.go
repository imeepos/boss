package adminapi

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/monthly"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// monthlyRegionsHandler GET /monthly/regions:区域白名单(51 Barangay,含 active)。
func monthlyRegionsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Monthly.ListRegions(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// monthlyListFilter 解析 month/region/page/pageSize。
func monthlyListFilter(c *gin.Context) monthly.Filter {
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("pageSize"))
	return monthly.Filter{Month: c.Query("month"), Region: c.Query("region"), Page: page, PageSize: size}
}

// monthlyListHandler GET /monthly/{table}:月×区域列表(响应含派生列)。
func monthlyListHandler(a *app.Application, kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		f := monthlyListFilter(c)
		ctx := c.Request.Context()
		switch kind {
		case monthly.TableUserRevenue:
			r, err := a.Monthly.ListUserRevenue(ctx, f)
			monthlyRespondPage(c, r, err)
		case monthly.TableNetworkDelivery:
			r, err := a.Monthly.ListNetworkDelivery(ctx, f)
			monthlyRespondPage(c, r, err)
		case monthly.TableFinanceCost:
			r, err := a.Monthly.ListFinanceCost(ctx, f)
			monthlyRespondPage(c, r, err)
		default:
			respondErr(c, monthly.ErrUnknownTable)
		}
	}
}

// monthlyRespondPage 统一分页响应(泛型信封直出)。
func monthlyRespondPage[T any](c *gin.Context, r monthly.PageResult[T], err error) {
	if err != nil {
		respondErr(c, err)
		return
	}
	respond(c, apitypes.CodeOK, r)
}

// 月度单行 upsert 请求体(派生列刻意无字段:服务端计算,外部传入一律无效)。
type monthlyUserRevenueReq struct {
	Month             string `json:"month" binding:"required"`
	Region            string `json:"region" binding:"required"`
	OpeningActive     *int64 `json:"openingActive"`
	NewUsers          *int64 `json:"newUsers"`
	ChurnedUsers      *int64 `json:"churnedUsers"`
	AdjustedUsers     *int64 `json:"adjustedUsers"`
	BroadbandRevenue  *int64 `json:"broadbandRevenue"`
	ValueAddedRevenue *int64 `json:"valueAddedRevenue"`
	OnetimeCharge     *int64 `json:"onetimeCharge"`
	DiscountAmount    *int64 `json:"discountAmount"`
	RefundReversal    *int64 `json:"refundReversal"`
}

type monthlyNetworkDeliveryReq struct {
	Month             string `json:"month" binding:"required"`
	Region            string `json:"region" binding:"required"`
	InstallRequests   *int64 `json:"installRequests"`
	OntimeCompletions *int64 `json:"ontimeCompletions"`
	PortsDeployed     *int64 `json:"portsDeployed"`
	PortsActive       *int64 `json:"portsActive"`
	FaultReports      *int64 `json:"faultReports"`
	RepairHours       *int64 `json:"repairHours"`
}

type monthlyFinanceCostReq struct {
	Month            string `json:"month" binding:"required"`
	Region           string `json:"region" binding:"required"`
	InvoicedAmount   *int64 `json:"invoicedAmount"`
	CollectedAmount  *int64 `json:"collectedAmount"`
	ReceivableEnding *int64 `json:"receivableEnding"`
	DirectCost       *int64 `json:"directCost"`
	FixedCost        *int64 `json:"fixedCost"`
	CapexInvest      *int64 `json:"capexInvest"`
}

// i64 指针解引(缺省 0,PUT 全量覆盖语义与导入一致)。
func i64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// monthlyUpsertHandler PUT /monthly/{table}:单行 upsert(校验同导入)。
func monthlyUpsertHandler(a *app.Application, kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var err error
		var key string
		switch kind {
		case monthly.TableUserRevenue:
			var req monthlyUserRevenueReq
			if !httpx.BindAndValidate(c, &req) {
				return
			}
			key = req.Month + "/" + req.Region
			err = a.Monthly.UpsertUserRevenue(ctx, monthly.UserRevenue{Month: req.Month, Region: req.Region,
				OpeningActive: i64(req.OpeningActive), NewUsers: i64(req.NewUsers), ChurnedUsers: i64(req.ChurnedUsers),
				AdjustedUsers: i64(req.AdjustedUsers), BroadbandRevenue: i64(req.BroadbandRevenue),
				ValueAddedRevenue: i64(req.ValueAddedRevenue), OnetimeCharge: i64(req.OnetimeCharge),
				DiscountAmount: i64(req.DiscountAmount), RefundReversal: i64(req.RefundReversal)})
		case monthly.TableNetworkDelivery:
			var req monthlyNetworkDeliveryReq
			if !httpx.BindAndValidate(c, &req) {
				return
			}
			key = req.Month + "/" + req.Region
			err = a.Monthly.UpsertNetworkDelivery(ctx, monthly.NetworkDelivery{Month: req.Month, Region: req.Region,
				InstallRequests: i64(req.InstallRequests), OntimeCompletions: i64(req.OntimeCompletions),
				PortsDeployed: i64(req.PortsDeployed), PortsActive: i64(req.PortsActive),
				FaultReports: i64(req.FaultReports), RepairHours: i64(req.RepairHours)})
		case monthly.TableFinanceCost:
			var req monthlyFinanceCostReq
			if !httpx.BindAndValidate(c, &req) {
				return
			}
			key = req.Month + "/" + req.Region
			err = a.Monthly.UpsertFinanceCost(ctx, monthly.FinanceCost{Month: req.Month, Region: req.Region,
				InvoicedAmount: i64(req.InvoicedAmount), CollectedAmount: i64(req.CollectedAmount),
				ReceivableEnding: i64(req.ReceivableEnding), DirectCost: i64(req.DirectCost),
				FixedCost: i64(req.FixedCost), CapexInvest: i64(req.CapexInvest)})
		default:
			err = monthly.ErrUnknownTable
		}
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "monthly.upsert", kind, key, nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// monthlySummaryHandler GET /monthly/summary?month=:汇总 KPI(分母 0 → null)。
func monthlySummaryHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		sum, err := a.Monthly.Summary(c.Request.Context(), c.Query("month"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, sum)
	}
}
