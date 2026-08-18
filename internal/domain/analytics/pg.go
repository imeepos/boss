package analytics

// PGStore AnalyticsService 的 PostgreSQL 实现(阶段9,派生聚合)。
// 口径(公开可解释):
//   端口利用率   = USED+RESERVED 端口 / 非 DISABLED 端口
//   装机转化率   = DONE 订单 / 非 CANCELLED 订单
//   单用户维护成本 = Σ设备故障次数 × 维护单价 / 在服客户数
//   资产健康度   = AVG(device_maintenances.health_score)
//   区域 ROI     = 区域已收款 / (扩容端口数 × 单端口成本)

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// dbtx 最小数据库接口(*pgxpool.Pool / pgxmock 均满足)。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 分析聚合实现。
type PGStore struct {
	db            dbtx
	maintUnitCost float64 // 单次维护成本(元)
	portUnitCost  float64 // 单端口扩容成本(元)
}

// NewPGStore 构造;成本单价可配(缺省 50/800)。
func NewPGStore(db dbtx, maintUnitCost, portUnitCost float64) *PGStore {
	if maintUnitCost <= 0 {
		maintUnitCost = 50
	}
	if portUnitCost <= 0 {
		portUnitCost = 800
	}
	return &PGStore{db: db, maintUnitCost: maintUnitCost, portUnitCost: portUnitCost}
}

// FiveIndicators 五大指标 + 区域 ROI 明细。
func (s *PGStore) FiveIndicators(ctx context.Context) ([]Indicator, []RegionROI, error) {
	var portUsed, portTotal, ordDone, ordEff, faults, activeCust int64
	var health pgtype.Float8
	err := s.db.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM ports WHERE status IN ('USED','RESERVED')),
		  (SELECT count(*) FROM ports WHERE status <> 'DISABLED'),
		  (SELECT count(*) FROM orders WHERE status = 'DONE'),
		  (SELECT count(*) FROM orders WHERE status <> 'CANCELLED'),
		  (SELECT COALESCE(SUM(fault_count),0) FROM device_maintenances),
		  (SELECT count(*) FROM customers WHERE service_status = 'ACTIVE'),
		  (SELECT AVG(health_score) FROM device_maintenances)`).
		Scan(&portUsed, &portTotal, &ordDone, &ordEff, &faults, &activeCust, &health)
	if err != nil {
		return nil, nil, fmt.Errorf("analytics: indicators: %w", err)
	}
	rois, revenue, invest, err := s.regionROI(ctx)
	if err != nil {
		return nil, nil, err
	}
	ind := []Indicator{
		ratioInd("portUtilization", "端口利用率", portUsed, portTotal),
		ratioInd("installConversion", "装机转化率", ordDone, ordEff),
		s.indMaintCost(faults, activeCust),
		s.indHealth(health),
		roiInd(revenue, invest),
	}
	return ind, rois, nil
}

// ratioInd 比率类指标(分母 0 时值 0,口径仍展示分子/分母)。
func ratioInd(key, name string, num, den int64) Indicator {
	v := 0.0
	if den > 0 {
		v = float64(num) / float64(den)
	}
	return Indicator{Key: key, Name: name, Value: v, Detail: fmt.Sprintf("%d/%d", num, den)}
}

// indMaintCost 单用户维护成本。
func (s *PGStore) indMaintCost(faults, activeCust int64) Indicator {
	v := 0.0
	if activeCust > 0 {
		v = float64(faults) * s.maintUnitCost / float64(activeCust)
	}
	return Indicator{
		Key: "maintenanceCostPerUser", Name: "单用户维护成本(元)",
		Value: v, Detail: fmt.Sprintf("%d次×%.0f元/%d户", faults, s.maintUnitCost, activeCust),
	}
}

// indHealth 资产健康度(无数据=0)。
func (s *PGStore) indHealth(h pgtype.Float8) Indicator {
	v := 0.0
	detail := "无维护档案"
	if h.Valid {
		v = h.Float64
		detail = "AVG(health_score)"
	}
	return Indicator{Key: "assetHealth", Name: "资产健康度评分", Value: v, Detail: detail}
}

// roiInd 总 ROI。
func roiInd(revenue, invest float64) Indicator {
	v := 0.0
	if invest > 0 {
		v = revenue / invest
	}
	return Indicator{
		Key: "regionROI", Name: "区域投资回报率",
		Value: v, Detail: fmt.Sprintf("收入%.2f元/投资%.2f元", revenue, invest),
	}
}

// regionROI 区域维度下钻:收入(payments SUCCESS 经 bills 区域快照)与扩容投资。
func (s *PGStore) regionROI(ctx context.Context) ([]RegionROI, float64, float64, error) {
	rows, err := s.db.Query(ctx, `
		WITH rev AS (
		  SELECT b.region_id, COALESCE(SUM(p.amount),0) AS amt
		  FROM payments p JOIN bills b ON b.id = p.bill_id
		  WHERE p.status = 'SUCCESS' GROUP BY b.region_id
		), inv AS (
		  SELECT region_id, SUM(expected_ports) * $1::numeric AS amt
		  FROM expansions GROUP BY region_id
		)
		SELECT r.id, r.name, COALESCE(rev.amt,0), COALESCE(inv.amt,0)
		FROM regions r
		LEFT JOIN rev ON rev.region_id = r.id
		LEFT JOIN inv ON inv.region_id = r.id
		ORDER BY r.id`, s.portUnitCost)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("analytics: region roi: %w", err)
	}
	defer rows.Close()
	out := make([]RegionROI, 0)
	var totRev, totInv float64
	for rows.Next() {
		var c RegionROI
		var amt float64
		if err := rows.Scan(&c.RegionID, &c.RegionName, &c.Revenue, &amt); err != nil {
			return nil, 0, 0, fmt.Errorf("analytics: scan roi: %w", err)
		}
		c.Investment = amt
		if amt > 0 {
			c.ROI = c.Revenue / amt
		}
		totRev += c.Revenue
		totInv += amt
		out = append(out, c)
	}
	return out, totRev, totInv, rows.Err()
}

// Heatmap 按楼栋(5)/小区(4)聚合端口利用率,高利用率在前。
func (s *PGStore) Heatmap(ctx context.Context) ([]HeatCell, error) {
	rows, err := s.db.Query(ctx, `
		SELECT a.id, a.name, a.level,
		       count(p.id) AS total,
		       count(p.id) FILTER (WHERE p.status IN ('USED','RESERVED')) AS used
		FROM addresses a
		JOIN ports p ON p.address_id = a.id
		WHERE a.level IN (4,5)
		GROUP BY a.id, a.name, a.level
		ORDER BY (count(p.id) FILTER (WHERE p.status IN ('USED','RESERVED')))::float
		         / NULLIF(count(p.id),0) DESC NULLS LAST, a.id`)
	if err != nil {
		return nil, fmt.Errorf("analytics: heatmap: %w", err)
	}
	defer rows.Close()
	out := make([]HeatCell, 0)
	for rows.Next() {
		var c HeatCell
		if err := rows.Scan(&c.AddressID, &c.Name, &c.Level, &c.PortsTotal, &c.PortsUsed); err != nil {
			return nil, fmt.Errorf("analytics: scan heat: %w", err)
		}
		if c.PortsTotal > 0 {
			c.Utilization = float64(c.PortsUsed) / float64(c.PortsTotal)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// MaintenanceList 维护一张表:优先级(MUST_REPLACE>SUGGEST>WATCH)→健康度→故障率→年限。
func (s *PGStore) MaintenanceList(ctx context.Context) ([]MaintenanceItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT device_no, device_type, health_score, fault_count, age_years, reason, priority
		FROM device_maintenances
		ORDER BY CASE priority
		           WHEN 'MUST_REPLACE' THEN 0 WHEN 'SUGGEST' THEN 1 ELSE 2 END,
		         health_score ASC, fault_count DESC, age_years DESC`)
	if err != nil {
		return nil, fmt.Errorf("analytics: maintenance: %w", err)
	}
	defer rows.Close()
	out := make([]MaintenanceItem, 0)
	for rows.Next() {
		var m MaintenanceItem
		if err := rows.Scan(&m.DeviceNo, &m.DeviceType, &m.HealthScore, &m.FaultCount, &m.AgeYears, &m.Reason, &m.Priority); err != nil {
			return nil, fmt.Errorf("analytics: scan maint: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
