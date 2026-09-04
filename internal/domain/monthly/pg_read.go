package monthly

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// 列清单(顺序=Scan 顺序;user_revenue 尾追两个派生生成列)。
const (
	urCols = "month, region, opening_active, new_users, churned_users, adjusted_users, " +
		"broadband_revenue, value_added_revenue, onetime_charge, discount_amount, refund_reversal, " +
		"closing_active, total_revenue, updated_at"
	ndCols      = "month, region, install_requests, ontime_completions, ports_deployed, ports_active, fault_reports, repair_hours, updated_at"
	fcCols      = "month, region, invoiced_amount, collected_amount, receivable_ending, direct_cost, fixed_cost, capex_invest, updated_at"
	tblUserRev  = "monthly_user_revenue"
	tblNetDeliv = "monthly_network_delivery"
	tblFinCost  = "monthly_finance_cost"
)

// normalizeFilter 分页缺省(1/20,上限 200)。
func normalizeFilter(f Filter) Filter {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	if f.PageSize > 200 {
		f.PageSize = 200
	}
	return f
}

// pageQuery 共用分页查询:month/region 过滤 + 计数 + 行集(排序 month,region)。
// 表名/列名全部来自包内常量,无外部拼注入面。
func (s *PGStore) pageQuery(ctx context.Context, table, cols string, f Filter) (pgx.Rows, int, error) {
	f = normalizeFilter(f)
	const where = " WHERE ($1 = '' OR month = $1) AND ($2 = '' OR region = $2)"
	args := []any{f.Month, f.Region}
	var total int
	if err := s.db.QueryRow(ctx, "SELECT count(*) FROM "+table+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("monthly: count %s: %w", table, err)
	}
	q := "SELECT " + cols + " FROM " + table + where +
		" ORDER BY month, region LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	rows, err := s.db.Query(ctx, q, append(args, f.PageSize, (f.Page-1)*f.PageSize)...)
	if err != nil {
		return nil, 0, fmt.Errorf("monthly: list %s: %w", table, err)
	}
	return rows, total, nil
}

// scanStamp 共用 updated_at 扫描。
func scanStamp(ts pgtype.Timestamptz) string {
	if ts.Valid {
		return ts.Time.Format(time.RFC3339)
	}
	return ""
}

// ListUserRevenue 用户与收入列表(响应含库端派生列)。
func (s *PGStore) ListUserRevenue(ctx context.Context, f Filter) (PageResult[UserRevenue], error) {
	rows, total, err := s.pageQuery(ctx, tblUserRev, urCols, f)
	if err != nil {
		return PageResult[UserRevenue]{}, err
	}
	defer rows.Close()
	items := make([]UserRevenue, 0)
	for rows.Next() {
		var m, region string
		var v [9]int64
		var closing, totalRev int64
		var ts pgtype.Timestamptz
		if err := rows.Scan(&m, &region, &v[0], &v[1], &v[2], &v[3], &v[4], &v[5], &v[6], &v[7], &v[8], &closing, &totalRev, &ts); err != nil {
			return PageResult[UserRevenue]{}, fmt.Errorf("monthly: scan user revenue: %w", err)
		}
		item := userRevenueFromRow(csvRow{Month: m, Region: region, Values: v[:]}, scanStamp(ts))
		item.ClosingActive, item.TotalRevenue = closing, totalRev // 库端生成列为准
		items = append(items, item)
	}
	return PageResult[UserRevenue]{Items: items, Total: total, Page: normalizeFilter(f).Page, PageSize: normalizeFilter(f).PageSize}, rows.Err()
}

// ListNetworkDelivery 网络与交付列表。
func (s *PGStore) ListNetworkDelivery(ctx context.Context, f Filter) (PageResult[NetworkDelivery], error) {
	rows, total, err := s.pageQuery(ctx, tblNetDeliv, ndCols, f)
	if err != nil {
		return PageResult[NetworkDelivery]{}, err
	}
	defer rows.Close()
	items := make([]NetworkDelivery, 0)
	for rows.Next() {
		var m, region string
		var v [6]int64
		var ts pgtype.Timestamptz
		if err := rows.Scan(&m, &region, &v[0], &v[1], &v[2], &v[3], &v[4], &v[5], &ts); err != nil {
			return PageResult[NetworkDelivery]{}, fmt.Errorf("monthly: scan network delivery: %w", err)
		}
		items = append(items, networkDeliveryFromRow(csvRow{Month: m, Region: region, Values: v[:]}, scanStamp(ts)))
	}
	return PageResult[NetworkDelivery]{Items: items, Total: total, Page: normalizeFilter(f).Page, PageSize: normalizeFilter(f).PageSize}, rows.Err()
}

// ListFinanceCost 财务与成本列表。
func (s *PGStore) ListFinanceCost(ctx context.Context, f Filter) (PageResult[FinanceCost], error) {
	rows, total, err := s.pageQuery(ctx, tblFinCost, fcCols, f)
	if err != nil {
		return PageResult[FinanceCost]{}, err
	}
	defer rows.Close()
	items := make([]FinanceCost, 0)
	for rows.Next() {
		var m, region string
		var v [6]int64
		var ts pgtype.Timestamptz
		if err := rows.Scan(&m, &region, &v[0], &v[1], &v[2], &v[3], &v[4], &v[5], &ts); err != nil {
			return PageResult[FinanceCost]{}, fmt.Errorf("monthly: scan finance cost: %w", err)
		}
		items = append(items, financeCostFromRow(csvRow{Month: m, Region: region, Values: v[:]}, scanStamp(ts)))
	}
	return PageResult[FinanceCost]{Items: items, Total: total, Page: normalizeFilter(f).Page, PageSize: normalizeFilter(f).PageSize}, rows.Err()
}
