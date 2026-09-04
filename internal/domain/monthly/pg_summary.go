package monthly

import (
	"context"
	"fmt"
)

// Summary 月度汇总 KPI(口径见 fields.md 月度填报节):
// 及时完工率=及时完工÷装机申请;端口利用率=在用÷部署;回款率=实际回款÷开票;
// ARPU=主营总收入÷期末在用。分母为 0 时该率 nil(JSON null)不报错。
func (s *PGStore) Summary(ctx context.Context, month string) (Summary, error) {
	if month != "" {
		if err := ValidateMonth(month); err != nil {
			return Summary{}, err
		}
	}
	const cond = " WHERE ($1 = '' OR month = $1)"
	q := "SELECT" +
		" COALESCE((SELECT SUM(closing_active) FROM " + tblUserRev + cond + "), 0)," +
		" COALESCE((SELECT SUM(total_revenue) FROM " + tblUserRev + cond + "), 0)," +
		" COALESCE((SELECT SUM(install_requests) FROM " + tblNetDeliv + cond + "), 0)," +
		" COALESCE((SELECT SUM(ontime_completions) FROM " + tblNetDeliv + cond + "), 0)," +
		" COALESCE((SELECT SUM(ports_deployed) FROM " + tblNetDeliv + cond + "), 0)," +
		" COALESCE((SELECT SUM(ports_active) FROM " + tblNetDeliv + cond + "), 0)," +
		" COALESCE((SELECT SUM(invoiced_amount) FROM " + tblFinCost + cond + "), 0)," +
		" COALESCE((SELECT SUM(collected_amount) FROM " + tblFinCost + cond + "), 0)"
	var closing, revenue, requests, ontime, deployed, active, invoiced, collected int64
	if err := s.db.QueryRow(ctx, q, month).Scan(&closing, &revenue, &requests, &ontime, &deployed, &active, &invoiced, &collected); err != nil {
		return Summary{}, fmt.Errorf("monthly: summary: %w", err)
	}
	return Summary{
		Month:              month,
		ClosingActiveTotal: closing,
		TotalRevenueTotal:  revenue,
		OntimeRate:         Ratio(ontime, requests),
		PortUtilization:    Ratio(active, deployed),
		CollectionRate:     Ratio(collected, invoiced),
		Arpu:               Ratio(revenue, closing),
	}, nil
}
