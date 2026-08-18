package analytics

// StarRocksStore AnalyticsService 的 StarRocks 实现(MySQL 协议,阶段9 宽表)。
// 宽表 DDL:deployments/olap/starrocks-init.sql;口径与 PGStore 一致,数据由 ETL 写入。

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// NewStarRocksStore 构造(dsn 如 root@tcp(192.168.0.102:29030)/,库名缺省补 boss_olap)。
func NewStarRocksStore(dsn string, maintUnitCost, portUnitCost float64) (*StarRocksStore, error) {
	if !strings.Contains(dsn, "/boss_olap") {
		if strings.HasSuffix(dsn, "/") {
			dsn += "boss_olap"
		} else if strings.HasSuffix(dsn, "/?") || strings.HasSuffix(dsn, "/") {
			dsn = strings.TrimSuffix(dsn, "?") + "boss_olap?"
		}
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("analytics: starrocks open: %w", err)
	}
	s := &StarRocksStore{db: db, maintUnitCost: maintUnitCost, portUnitCost: portUnitCost}
	s.normalizeCosts()
	return s, nil
}

// StarRocksStore 宽表实现。
type StarRocksStore struct {
	db            *sql.DB
	maintUnitCost float64
	portUnitCost  float64
}

func (s *StarRocksStore) normalizeCosts() {
	if s.maintUnitCost <= 0 {
		s.maintUnitCost = 50
	}
	if s.portUnitCost <= 0 {
		s.portUnitCost = 800
	}
}

// Ping 连通性检查(装配层选后端时使用)。
func (s *StarRocksStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close 关闭连接池。
func (s *StarRocksStore) Close() error { return s.db.Close() }

// FiveIndicators 五大指标 + 区域 ROI(单查询聚合宽表,口径同 PGStore)。
func (s *StarRocksStore) FiveIndicators(ctx context.Context) ([]Indicator, []RegionROI, error) {
	var portUsed, portTotal, ordDone, ordEff, activeCust int64
	var faults sql.NullInt64
	var health sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
		SELECT
		  (SELECT COALESCE(SUM(ports_used),0) FROM wide_port_heat),
		  (SELECT COALESCE(SUM(ports_total),0) FROM wide_port_heat),
		  (SELECT COALESCE(MAX(orders_done),0) FROM wide_order_flow),
		  (SELECT COALESCE(MAX(orders_effective),0) FROM wide_order_flow),
		  (SELECT COALESCE(SUM(fault_count),0) FROM wide_device_health),
		  (SELECT COALESCE(MAX(active_customers),0) FROM wide_order_flow),
		  (SELECT AVG(health_score) FROM wide_device_health)`).
		Scan(&portUsed, &portTotal, &ordDone, &ordEff, &faults, &activeCust, &health)
	if err != nil {
		return nil, nil, fmt.Errorf("analytics: starrocks indicators: %w", err)
	}
	rois, totRev, totInv, err := s.regionROI(ctx)
	if err != nil {
		return nil, nil, err
	}
	return []Indicator{
		ratioInd("portUtilization", "端口利用率", portUsed, portTotal),
		ratioInd("installConversion", "装机转化率", ordDone, ordEff),
		s.indMaintCost(faults.Int64, activeCust),
		nullHealthInd(health),
		roiInd(totRev, totInv),
	}, rois, nil
}

// nullHealthInd 资产健康度(可空)。
func nullHealthInd(h sql.NullFloat64) Indicator {
	v, detail := 0.0, "无维护档案"
	if h.Valid {
		v, detail = h.Float64, "AVG(health_score)"
	}
	return Indicator{Key: "assetHealth", Name: "资产健康度评分", Value: v, Detail: detail}
}

// indMaintCost 单用户维护成本(口径同 PGStore)。
func (s *StarRocksStore) indMaintCost(faults, activeCust int64) Indicator {
	v := 0.0
	if activeCust > 0 {
		v = float64(faults) * s.maintUnitCost / float64(activeCust)
	}
	return Indicator{
		Key: "maintenanceCostPerUser", Name: "单用户维护成本(元)",
		Value: v, Detail: fmt.Sprintf("%d次×%.0f元/%d户", faults, s.maintUnitCost, activeCust),
	}
}

// regionROI 区域下钻明细 + 总收入/总投资。
func (s *StarRocksStore) regionROI(ctx context.Context) ([]RegionROI, float64, float64, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT region_id, region_name, COALESCE(revenue,0), COALESCE(investment,0)
		 FROM wide_region_roi ORDER BY region_id`)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("analytics: starrocks roi: %w", err)
	}
	defer rows.Close()
	out := make([]RegionROI, 0)
	var totRev, totInv float64
	for rows.Next() {
		var c RegionROI
		if err := rows.Scan(&c.RegionID, &c.RegionName, &c.Revenue, &c.Investment); err != nil {
			return nil, 0, 0, fmt.Errorf("analytics: scan roi: %w", err)
		}
		if c.Investment > 0 {
			c.ROI = c.Revenue / c.Investment
		}
		totRev += c.Revenue
		totInv += c.Investment
		out = append(out, c)
	}
	return out, totRev, totInv, rows.Err()
}

// Heatmap 热力(利用率降序)。
func (s *StarRocksStore) Heatmap(ctx context.Context) ([]HeatCell, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT address_id, name, level, ports_total, ports_used,
		       CASE WHEN ports_total > 0 THEN ports_used / ports_total ELSE 0 END AS util
		FROM wide_port_heat
		ORDER BY util DESC, address_id`)
	if err != nil {
		return nil, fmt.Errorf("analytics: starrocks heatmap: %w", err)
	}
	defer rows.Close()
	return scanHeat(rows)
}

// MaintenanceList 维护一张表(排序口径同 PGStore)。
func (s *StarRocksStore) MaintenanceList(ctx context.Context) ([]MaintenanceItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT device_no, device_type, health_score, fault_count, age_years, reason, priority
		FROM wide_device_health
		ORDER BY CASE priority WHEN 'MUST_REPLACE' THEN 0 WHEN 'SUGGEST' THEN 1 ELSE 2 END,
		         health_score ASC, fault_count DESC, age_years DESC`)
	if err != nil {
		return nil, fmt.Errorf("analytics: starrocks maintenance: %w", err)
	}
	defer rows.Close()
	return scanMaint(rows)
}
