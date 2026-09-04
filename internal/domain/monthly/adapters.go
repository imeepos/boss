package monthly

// typed↔csvRow 适配:量值顺序=CSV/DB 列;派生列只出现在响应,不进导入导出。

// userRevenueRow 实体 → 事实行。
func userRevenueRow(r UserRevenue) csvRow {
	return csvRow{Month: r.Month, Region: r.Region, Values: []int64{
		r.OpeningActive, r.NewUsers, r.ChurnedUsers, r.AdjustedUsers,
		r.BroadbandRevenue, r.ValueAddedRevenue, r.OnetimeCharge, r.DiscountAmount, r.RefundReversal}}
}

// userRevenueFromRow 事实行 → 实体(派生列服务端补算,忽略外部传入)。
func userRevenueFromRow(r csvRow, updatedAt string) UserRevenue {
	v := r.Values
	return UserRevenue{
		Month: r.Month, Region: r.Region,
		OpeningActive: v[0], NewUsers: v[1], ChurnedUsers: v[2], AdjustedUsers: v[3],
		ClosingActive:    ClosingActive(v[0], v[1], v[2], v[3]),
		BroadbandRevenue: v[4], ValueAddedRevenue: v[5], OnetimeCharge: v[6], DiscountAmount: v[7], RefundReversal: v[8],
		TotalRevenue: TotalRevenue(v[4], v[5], v[6], v[7], v[8]),
		UpdatedAt:    updatedAt,
	}
}

func networkDeliveryRow(r NetworkDelivery) csvRow {
	return csvRow{Month: r.Month, Region: r.Region, Values: []int64{
		r.InstallRequests, r.OntimeCompletions, r.PortsDeployed, r.PortsActive, r.FaultReports, r.RepairHours}}
}

func networkDeliveryFromRow(r csvRow, updatedAt string) NetworkDelivery {
	v := r.Values
	return NetworkDelivery{Month: r.Month, Region: r.Region,
		InstallRequests: v[0], OntimeCompletions: v[1], PortsDeployed: v[2], PortsActive: v[3], FaultReports: v[4], RepairHours: v[5],
		UpdatedAt: updatedAt}
}

func financeCostRow(r FinanceCost) csvRow {
	return csvRow{Month: r.Month, Region: r.Region, Values: []int64{
		r.InvoicedAmount, r.CollectedAmount, r.ReceivableEnding, r.DirectCost, r.FixedCost, r.CapexInvest}}
}

func financeCostFromRow(r csvRow, updatedAt string) FinanceCost {
	v := r.Values
	return FinanceCost{Month: r.Month, Region: r.Region,
		InvoicedAmount: v[0], CollectedAmount: v[1], ReceivableEnding: v[2], DirectCost: v[3], FixedCost: v[4], CapexInvest: v[5],
		UpdatedAt: updatedAt}
}

// validateFact 单行校验(PUT 与导入同规):月份/区域/非负整数。
func validateFact(kind string, r csvRow) error {
	if _, err := specOf(kind); err != nil {
		return err
	}
	if err := ValidateMonth(r.Month); err != nil {
		return err
	}
	if r.Region == "" {
		return ErrInvalidInput
	}
	for _, v := range r.Values {
		if v < 0 {
			return ErrInvalidInput
		}
	}
	return nil
}
